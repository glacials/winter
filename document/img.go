package document // import "twos.dev/winter/document"

import (
	"errors"
	"fmt"
	"hash/fnv"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/nickalie/go-webpbin"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/rwcarlsen/goexif/tiff"
	"golang.org/x/image/draw"
)

type EXIF struct {
	Aperture    float64
	Camera      *Gear
	FocalLength float64
	ISO         string
	// Lens holds information about the lens used for the photo.
	// If the photo EXIF data has no or insufficient lens information,
	// Lens is nil.
	Lens         *Gear
	ShutterSpeed string
	TakenAt      time.Time
}

type img struct {
	EXIF

	Alt        string
	Thumbnails thumbnails
	// SourcePath is the path to the image this gallery document was built around.
	// It is relative to the repository root.
	SourcePath string
	// WebPath is the path component of the URL to the image as it will exist after building.
	WebPath string

	cfg    *Config
	logger *slog.Logger
	photo  image.Image
}

type thumbnail struct {
	// Height is the height of the thumbnail.
	Height int
	// Width is the width of the thumbnail.
	Width int

	// WebPath is the path component of the URL where the thumbnail will ultimately be placed.
	// It is equivalent to the thumbnail's path relative to dist.
	WebPath string
}

type thumbnails []*thumbnail

func (t thumbnails) Len() int {
	return len(t)
}

func (t thumbnails) Less(i, j int) bool {
	return t[i].Width < t[j].Width
}

func (t thumbnails) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// NewIMG returns a struct that represents an image to be built.
// The returned value implements [Document].
func NewIMG(logger *slog.Logger, src string, cfg *Config) (*img, error) {
	relpath, err := filepath.Rel("src", src)
	if err != nil {
		return nil, fmt.Errorf("can't get relpath for photo `%s`: %w", src, err)
	}
	return &img{
		SourcePath: src,
		WebPath:    fmt.Sprintf("%s.webp", strings.TrimSuffix(relpath, filepath.Ext(relpath))),

		cfg:    cfg,
		logger: logger,
	}, nil
}

// LoadEXIF populates im's EXIF information without decoding the entire image.
func (im *img) LoadEXIF() error {
	f, err := os.Open(im.SourcePath)
	if err != nil {
		return fmt.Errorf("can't read %q: %w", im.SourcePath, err)
	}
	defer f.Close()
	if err := im.loadEXIF(f); err != nil {
		return fmt.Errorf("cannot get camera for %q: %w", im.SourcePath, err)
	}
	return nil
}

func (im *img) Load(r io.Reader) error {
	if err := im.loadEXIF(r); err != nil {
		return fmt.Errorf(
			"cannot get camera for %q: %w",
			im.SourcePath,
			err,
		)
	}
	srcf, err := os.Open(im.SourcePath)
	if err != nil {
		return fmt.Errorf("can't read %q: %w", im.SourcePath, err)
	}
	defer srcf.Close()
	srcPhoto, err := jpeg.Decode(srcf)
	if err != nil {
		return fmt.Errorf(
			"cannot decode photo %q (maybe not an image?): %w",
			im.SourcePath,
			err,
		)
	}
	im.photo = srcPhoto
	thumbdir := filepath.Dir(strings.Replace(
		filepath.Join(im.cfg.Dist, im.WebPath),
		filepath.FromSlash("/img/"),
		filepath.FromSlash("/img/thumb/"),
		1,
	))
	if err := im.intuitThumbnails(srcPhoto, im.SourcePath, thumbdir); err != nil {
		return err
	}
	return nil
}

func (im *img) Render(w io.Writer) error {
	if err := webpbin.Encode(w, im.photo); err != nil {
		return fmt.Errorf("cannot encode source image %q to WebP: %w", im.SourcePath, err)
	}
	if err := im.thumbnails(im.SourcePath); err != nil {
		return fmt.Errorf("can't generate thumbnails: %w", err)
	}
	return nil
}

func exifFractionToDecimal(
	x *exif.Exif,
	field exif.FieldName,
) (float64, error) {
	fraction, err := x.Get(field)
	if err != nil {
		return 0, fmt.Errorf("can't get field %s: %w", field, err)
	}
	parts := strings.Split(
		strings.Replace(fraction.String(), "\"", "", 2),
		"/",
	)
	numer, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf(
			"can't convert %s (numerator of %s, %s) to int: %w",
			parts[0],
			field,
			fraction,
			err,
		)
	}
	denom, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf(
			"can't convert %s (denominator of %s, %s) to int: %w",
			parts[0],
			field,
			fraction,
			err,
		)
	}

	return float64(numer) / float64(denom), nil
}

// generatedPhotosAreFresh returns true if and only if the file at src has the same content as the last time this function was called.
//
// The XDG cache directory for Winter is used to store state.
// To empty it, run winter clean.
func (d *img) generatedPhotosAreFresh(src string) (bool, error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return false, fmt.Errorf("can't open photo at path `%s`: %w", src, err)
	}
	defer sourceFile.Close()
	buf, err := io.ReadAll(sourceFile)
	if err != nil {
		return false, fmt.Errorf("cannot hash file %q: %w", src, err)
	}

	hash := fnv.New32()
	_, err = hash.Write(buf)
	if err != nil {
		return false, fmt.Errorf("cannot compute hash for %q: %w", src, err)
	}

	sumPath, err := xdg.CacheFile(fmt.Sprintf("%s.sum", filepath.Join(AppName, "generated", "img", filepath.Base(src))))
	if err != nil {
		return false, fmt.Errorf("cannot find Winter cache: %w", err)
	}
	newSum := hash.Sum32()
	oldSum, err := os.ReadFile(sumPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("cannot read old sum at %q: %w", oldSum, err)
		}
	}
	if fmt.Sprintf("%d", newSum) == string(oldSum) {
		return true, nil
	}
	if err := os.MkdirAll(filepath.Dir(sumPath), 0o755); err != nil {
		return false, fmt.Errorf("cann't make thumbnail directory for sums %q: %w", filepath.Dir(sumPath), err)
	}
	if err := os.WriteFile(sumPath, []byte(fmt.Sprintf("%d", newSum)), 0o644); err != nil {
		return false, fmt.Errorf("cannot write hash for %q: %w", src, err)
	}
	return false, nil
}

// loadEXIF extracts the EXIF string
// (including lens, etc.)
// and timestamp from the image at the given path.
func (im *img) loadEXIF(r io.Reader) error {
	exif.RegisterParsers(mknote.All...)
	x, err := exif.Decode(r)
	if err != nil {
		return fmt.Errorf("cannot read exif data: %w", err)
	}
	camera, err := im.findGear(x, exif.Make, exif.Model)
	if err != nil {
		return fmt.Errorf("cannot get camera: %w", err)
	}
	exposure, err := x.Get(exif.ExposureTime)
	if err != nil {
		return fmt.Errorf("cannot get exposure: %w", err)
	}
	fnum, err := exifFractionToDecimal(x, exif.FNumber)
	if err != nil {
		return fmt.Errorf("cannot get focal length: %w", err)
	}
	focalLength, err := exifFractionToDecimal(x, exif.FocalLength)
	if err != nil {
		return fmt.Errorf("cannot get focal length: %w", err)
	}
	iso, err := x.Get(exif.ISOSpeedRatings)
	if err != nil {
		return fmt.Errorf("cannot get ISO: %w", err)
	}
	lens, err := im.findGear(x, exif.LensMake, exif.LensModel)
	if err != nil {
		return fmt.Errorf("cannot get lens: %w", err)
	}
	timestamp, err := x.DateTime()
	if err != nil {
		if errors.Is(err, exif.TagNotPresentError("")) {
			return fmt.Errorf(
				"photo has no EXIF timestamp... what do?",
			)
		}
		return fmt.Errorf("cannot get photo datetime: %w", err)
	}

	_, err = x.Get(exif.GPSInfoIFDPointer)
	if err == nil {
		// location data is set! no no no!
		panic(fmt.Sprintf("photo %s has location data! please strip it.", im.SourcePath))
	}

	im.EXIF = EXIF{
		Aperture:     fnum,
		Camera:       camera,
		FocalLength:  focalLength,
		ISO:          iso.String(),
		Lens:         lens,
		ShutterSpeed: strings.Replace(exposure.String(), "\"", "", 2),
		TakenAt:      timestamp,
	}
	return nil
}

// thumbnails makes WebP thumbnails of the photo srcPhoto located at srcPath.
// The thumbnails made are based on the specifications already in im.Thumbnails at call time.
//
// The file at src is read at least once every time this function is called,
// but the thumbnails are only regenerated if src has changed since their last generation.
func (im *img) thumbnails(srcPath string) error {
	im.logger.Debug(fmt.Sprintf("Creating thumbnails for %s.", srcPath))
	for _, thmb := range im.Thumbnails {
		if thmb.Width <= 0 || thmb.Height <= 0 {
			continue
		}
		im.logger.Debug(fmt.Sprintf("Created %dx%d thumbnail for %s.", thmb.Width, thmb.Height, srcPath))
		dstPhoto := image.NewRGBA(image.Rect(0, 0, thmb.Width, thmb.Height))

		draw.CatmullRom.Scale(
			dstPhoto,
			image.Rectangle{
				image.Point{0, 0},
				image.Point{thmb.Width, thmb.Height},
			},
			im.photo,
			image.Rectangle{image.Point{0, 0}, im.photo.Bounds().Size()},
			draw.Over,
			nil,
		)

		path := filepath.Join(im.cfg.Dist, thmb.WebPath)

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf(
				"cannot make thumbnail directory %q: %w",
				filepath.Dir(thmb.WebPath),
				err,
			)
		}

		destinationFile, err := os.Create(path)
		if err != nil {
			return fmt.Errorf(
				"cannot create thumbnail for %q: %w",
				srcPath,
				err,
			)
		}
		defer destinationFile.Close()

		if err := webpbin.Encode(destinationFile, dstPhoto); err != nil {
			return fmt.Errorf("cannot encode WebP thumbnail to %q: %w", thmb.WebPath, err)
		}
	}

	return nil
}

// intuitThumbnails decides how many and which thumbnails the image should have
// (whether or not they already exist)
// and replaces im.Thumbnails with a slice of them.
// It does not generate any thumbnails.
//
// The thumbnails decided upon have widths of powers of 2,
// from 1 until the largest width possible that is still smaller than the source image.
// Heights are automatically calculated to mantain aspect ratio.
//
// For example, a 500x500 image called foo.jpg would have thumbnails of sizes
// 1x1, 2x2, 4x4, 8x8, 16x16, 32x32, 64x64, 128x128, and 256x256.
// The thumbnail WebPaths would be
// foo.1x1.webp,
// foo.2x2.webp,
// and so on.
func (im *img) intuitThumbnails(srcPhoto image.Image, srcPath, thumbdir string) error {
	var thmbs thumbnails
	p := srcPhoto.Bounds().Size()
	for height := 1; height < p.X; height *= 2 {
		width := (height * p.X / p.Y) & -1
		if width <= 0 || height <= 0 {
			continue
		}
		destPath := filepath.Join(
			thumbdir,
			fmt.Sprintf("%s.%dx%d.webp", strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath)), width, height),
		)
		webPath, err := filepath.Rel(im.cfg.Dist, destPath)
		if err != nil {
			return fmt.Errorf("cannot get relative path for thumbnail %q: %w", thumbdir, err)
		}
		thmbs = append(thmbs, &thumbnail{
			Height:  height,
			WebPath: webPath,
			Width:   width,
		})
	}
	im.Thumbnails = thmbs
	return nil
}

// findGear returns a Gear built from the given EXIF data.
//
// If the EXIF data contains no or insufficient info,
// (nil, nil) is returned.
//
// If the EXIF data is present but winter.yml does not index it,
// an error is returned.
func (im *img) findGear(x *exif.Exif, make, model exif.FieldName) (*Gear, error) {
	gearMake, err := x.Get(make)
	if err != nil {
		if errors.Is(err, exif.TagNotPresentError(make)) {
			return nil, nil
		}
		return nil, fmt.Errorf("can't get gear make: %w", err)
	}
	gearModel, err := x.Get(model)
	if err != nil {
		if errors.Is(err, exif.TagNotPresentError(model)) {
			return nil, nil
		}
		return nil, fmt.Errorf("can't get gear model: %w", err)
	}
	gearMakeStr := sanitizeEXIFField(gearMake)
	gearModelStr := sanitizeEXIFField(gearModel)
	g, ok := im.cfg.GearByString(gearMakeStr, gearModelStr)
	if !ok {
		return nil, fmt.Errorf("no such gear with make=%q and model=%q", gearMakeStr, gearModelStr)
	}
	return g, nil
}

// exifCutSet is a string containing all the individual characters,
// in no specific order,
// that should be cut from both ends of an EXIF field before processing it.
const exifCutSet = " \""

// sanitizeEXIFField pulls the value out of the given EXIF field,
// strips whitespace and quotes from its ends,
// and returns the result.
func sanitizeEXIFField(field *tiff.Tag) string {
	fieldVal := field.String()
	fieldVal = strings.Trim(fieldVal, exifCutSet)
	return fieldVal
}

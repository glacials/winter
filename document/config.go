package document // import "twos.dev/winter/document"

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/feeds"
	"gopkg.in/yaml.v3"
	"twos.dev/winter/cliutils"
)

const (
	configRelDir   = "winter"
	configFileName = "winter.yml"
)

// Config is a configuration for the Winter build.
type Config struct {
	// Author is the information for the website author.
	// This is used in metadata such as that of the RSS feed.
	Author feeds.Author `yaml:"author,omitempty"`
	// Debug is a flag that enables debug mode.
	Debug bool `yaml:"debug,omitempty"`
	// Development contains options specific to development.
	// They have no impact when building for production.
	Development struct {
		// URL is the base URL you will connect to while developing your website or Winter.
		// If blank, it defaults to "http://localhost:8100".
		URL string `yaml:"url,omitempty"`
	} `yaml:"development,omitempty"`
	// Description is the Description of the website.
	// This is used as metadata for the RSS feed.
	Description string `yaml:"description,omitempty"`
	// Dist is the location the site will be built into,
	// relative to the working directory.
	// After a build, this directory is suitable for deployment to the web as a set of static files.
	//
	// In other words, the path of any file in dist,
	// relative to dist,
	// is equivalent to the path component of the URL for that file.
	//
	// If blank, defaults to ./dist.
	Dist string `yaml:"dist,omitempty"`
	// Known helps the generated site follow the "Cool URIs don't change" rule
	// by remembering certain facts about what the site looks like,
	// and checking newly-generated sites against those facts.
	Known struct {
		// URIs holds the path to the known URIs file,
		// which Winter will generate, update, and maintain.
		//
		// You should commit this file.
		//
		// If unset, defaults to src/uris.txt.
		URIs string `yaml:"urls,omitempty"`
	} `yaml:"known,omitempty"`
	// Name is the name of the website.
	// This is used in various places in and out of templates.
	Name string `yaml:"name,omitempty"`
	// Gear is an array of Gear objects,
	// each describing a camera, lens, or other piece of gear
	// whose information can be extracted from EXIF data.
	//
	// Gear is used by Winter when processing photos to display photograph information,
	// and provide links to purchase gear used in its creation.
	Gear       []Gear `yaml:"gear,omitempty"`
	Production struct {
		// URL is the base URL you will connect to to view your deployed website
		// (e.g. twos.dev or one.twos.dev or twos.dev:6667).
		// This is used in various backlinks, like those in the RSS feed.
		//
		// Must not be blank.
		URL string `yaml:"url,omitempty"`
	} `yaml:"production,omitempty"`
	// Since is the year the website was established,
	// whether through Winter or otherwise.
	// This is used as metadata for the RSS feed,
	// and as a copyright notice when needed.
	Since int `yaml:"since,omitempty"`
	// Src is an additional list of directories to search for source files beyond ./src.
	Src []string `yaml:"srca,omitempty"`
}

type Gear struct {
	// Make is the user-readable brand that created this piece of gear.
	Make string `yaml:"make,omitempty"`
	// Model is the user-readable model for this piece of gear.
	Model string `yaml:"model,omitempty"`
	// Link is a URL
	// (beginning in https://)
	// at which a user can purchase or read more about this piece of gear.
	Link string `yaml:"link,omitempty"`
	// EXIF specifies what EXIF data a photograph must have in order to be identified as having been taken with this piece of gear.
	EXIF struct {
		// Make is the value a photograph's EXIF data must have in the "make" field,
		// whether camera or lens,
		// for the photograph to be considered as having been taken with this piece of gear.
		//
		// Before comparison,
		// Winter will strip all trailing and leading spaces from both the EXIF data field and from this field.
		// Then, a case-insensitive comparison is performed.
		Make string `yaml:"make,omitempty"`
		// Make is the value a photograph's EXIF data must have in the "model" field,
		// whether camera or lens,
		// for the photograph to be considered as having been taken with this piece of gear.
		//
		// Before comparison,
		// Winter will strip all trailing and leading spaces from both the EXIF data field and from this field.
		// Then, a case-insensitive comparison is performed.
		Model string `yaml:"model,omitempty"`
	} `yaml:"exif,omitempty"`
}

func NewConfig() (*Config, error) {
	var c Config
	p, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	f, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("No config file found. Run winter init to create one interactively.")
		}
	}
	if err := yaml.Unmarshal(f, &c); err != nil {
		return nil, err
	}
	if c.Development.URL == "" {
		c.Development.URL = "http://localhost:8100"
	}
	if c.Production.URL == "" {
		return nil, fmt.Errorf("production.url must be specified in winter.yml")
	}
	if c.Known.URIs == "" {
		c.Known.URIs = "src/uris.txt"
	}
	for _, g := range c.Gear {
		if g.Make == "" {
			return nil, fmt.Errorf("winter.yml: gear item with model=%q must have `make` attribute", g.Model)
		}
		if g.Model == "" {
			return nil, fmt.Errorf("winter.yml: gear item with make=%q must have `model` attribute", g.Make)
		}
		if g.Link == "" {
			return nil, fmt.Errorf("winter.yml: gear item with make=%q and model=%q must have `link` attribute, pointing to a web page for the gear item", g.Make, g.Model)
		}
	}
	for i := range c.Src {
		c.Src[i] = os.ExpandEnv(strings.ReplaceAll(c.Src[i], "~", "$HOME"))
	}
	return &c, nil
}

// GearByString returns the Gear item for the given make and model EXIF tag values.
// If none could be found, ok is false.
func (c *Config) GearByString(make, model string) (g *Gear, ok bool) {
	for _, gear := range c.Gear {
		if strings.EqualFold(gear.EXIF.Make, make) && strings.EqualFold(gear.EXIF.Model, model) {
			return &gear, true
		}
	}
	return nil, false
}

func (c *Config) Save() error {
	bytes, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	p, err := ConfigPath()
	if err != nil {
		return err
	}
	return os.WriteFile(p, bytes, fs.FileMode(os.O_WRONLY))
}

func InteractiveConfig() error {
	p, err := ConfigPath()
	if err != nil {
		return err
	}
	w, err := os.OpenFile(p, os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	var c Config
	c.Author.Name = cliutils.MustAsk("Author name:", "")
	c.Author.Email = cliutils.MustAsk("Author email:", "")
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

func ConfigPath() (string, error) {
	if _, err := os.Stat(configFileName); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return configFileName, nil
		}
	} else {
		return configFileName, nil
	}
	userCfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCfg, configRelDir, configFileName), nil
}

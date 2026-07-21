package document

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/bep/imagemeta"
)

const (
	iptcCoreNamespace = "http://iptc.org/std/Iptc4xmpCore/1.0/xmlns/"
	plusNamespace     = "http://ns.useplus.org/ldf/xmp/1.0/"
	rdfNamespace      = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	xmlNamespace      = "http://www.w3.org/XML/1998/namespace"
)

type photoXMP struct {
	Alt         string
	PurchaseURL string
}

type localizedText struct {
	lang  string
	value string
}

// loadPhotoXMP extracts Winter's portable gallery metadata from the first
// standard XMP packet embedded in a JPEG. Images without XMP are valid and
// return an empty photoXMP.
func loadPhotoXMP(r io.ReadSeeker) (photoXMP, error) {
	var metadata photoXMP
	err := imagemeta.Decode(imagemeta.Options{
		R:           r,
		ImageFormat: imagemeta.JPEG,
		Sources:     imagemeta.XMP,
		HandleXMP: func(r io.Reader) error {
			packet, err := io.ReadAll(r)
			if err != nil {
				return fmt.Errorf("cannot read XMP packet: %w", err)
			}
			metadata, err = parsePhotoXMP(bytes.TrimRight(packet, "\x00"))
			return err
		},
	})
	if err != nil {
		return photoXMP{}, fmt.Errorf("cannot decode XMP: %w", err)
	}
	return metadata, nil
}

// parsePhotoXMP reads the standards-based fields Winter exposes to gallery
// templates. It deliberately matches namespace URIs instead of XML prefixes.
func parsePhotoXMP(packet []byte) (photoXMP, error) {
	decoder := xml.NewDecoder(bytes.NewReader(packet))
	var (
		altItems         []localizedText
		attributeAlt     []localizedText
		purchaseURLs     []string
		depth            int
		altDepth         int
		altRootItemStart int
		altItemDepth     int
		altItemLang      string
		altItemText      strings.Builder
		altDirectText    strings.Builder
		licensorDepth    int
		licensorURLDepth int
		licensorURLText  strings.Builder
	)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return photoXMP{}, fmt.Errorf("cannot parse XMP XML: %w", err)
		}

		switch token := token.(type) {
		case xml.StartElement:
			depth++

			for _, attr := range token.Attr {
				if attr.Name.Space == iptcCoreNamespace && attr.Name.Local == "AltTextAccessibility" {
					attributeAlt = append(attributeAlt, localizedText{lang: "x-default", value: attr.Value})
				}
			}

			if token.Name.Space == iptcCoreNamespace && token.Name.Local == "AltTextAccessibility" {
				altDepth = depth
				altRootItemStart = len(altItems)
				altDirectText.Reset()
			}
			if altDepth > 0 && token.Name.Space == rdfNamespace && token.Name.Local == "li" {
				altItemDepth = depth
				altItemLang = ""
				altItemText.Reset()
				for _, attr := range token.Attr {
					if attr.Name.Space == xmlNamespace && attr.Name.Local == "lang" {
						altItemLang = attr.Value
						break
					}
				}
			}

			if token.Name.Space == plusNamespace && token.Name.Local == "Licensor" {
				licensorDepth = depth
			}
			if licensorDepth > 0 {
				for _, attr := range token.Attr {
					if attr.Name.Space == plusNamespace && attr.Name.Local == "LicensorURL" {
						purchaseURLs = append(purchaseURLs, attr.Value)
					}
				}
				if token.Name.Space == plusNamespace && token.Name.Local == "LicensorURL" {
					licensorURLDepth = depth
					licensorURLText.Reset()
				}
			}

		case xml.CharData:
			if altDepth > 0 {
				altDirectText.Write([]byte(token))
			}
			if altItemDepth > 0 {
				altItemText.Write([]byte(token))
			}
			if licensorURLDepth > 0 {
				licensorURLText.Write([]byte(token))
			}

		case xml.EndElement:
			if altItemDepth == depth {
				altItems = append(altItems, localizedText{lang: altItemLang, value: altItemText.String()})
				altItemDepth = 0
				altItemLang = ""
				altItemText.Reset()
			}
			if altDepth == depth {
				if len(altItems) == altRootItemStart {
					attributeAlt = append(attributeAlt, localizedText{lang: "x-default", value: altDirectText.String()})
				}
				altDepth = 0
				altDirectText.Reset()
			}
			if licensorURLDepth == depth {
				purchaseURLs = append(purchaseURLs, licensorURLText.String())
				licensorURLDepth = 0
				licensorURLText.Reset()
			}
			if licensorDepth == depth {
				licensorDepth = 0
			}
			depth--
		}
	}

	altItems = append(altItems, attributeAlt...)
	return photoXMP{
		Alt:         preferredLocalizedText(altItems),
		PurchaseURL: firstNonempty(purchaseURLs),
	}, nil
}

func preferredLocalizedText(items []localizedText) string {
	for _, preferred := range []func(string) bool{
		func(lang string) bool { return strings.EqualFold(lang, "x-default") },
		func(lang string) bool {
			lang = strings.ToLower(lang)
			return lang == "en" || strings.HasPrefix(lang, "en-")
		},
		func(string) bool { return true },
	} {
		for _, item := range items {
			if value := strings.TrimSpace(item.value); value != "" && preferred(item.lang) {
				return value
			}
		}
	}
	return ""
}

func firstNonempty(values []string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

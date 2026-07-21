package document

import (
	"bytes"
	"encoding/binary"
	"testing"

	"gotest.tools/v3/assert"
)

func TestParsePhotoXMP(t *testing.T) {
	tests := []struct {
		name         string
		packet       string
		wantAlt      string
		wantPurchase string
	}{
		{
			name: "prefers x-default and first nonempty licensor URL",
			packet: xmpPacket(`
<core:AltTextAccessibility>
  <rdf:Alt>
    <rdf:li xml:lang="de">Deutsch</rdf:li>
    <rdf:li xml:lang="en-US">English</rdf:li>
    <rdf:li xml:lang="x-default"> Default text </rdf:li>
  </rdf:Alt>
</core:AltTextAccessibility>
<plus:Licensor>
  <rdf:Bag>
    <rdf:li rdf:parseType="Resource"><plus:LicensorURL> </plus:LicensorURL></rdf:li>
    <rdf:li rdf:parseType="Resource"><plus:LicensorURL> https://example.com/first </plus:LicensorURL></rdf:li>
    <rdf:li rdf:parseType="Resource"><plus:LicensorURL>https://example.com/second</plus:LicensorURL></rdf:li>
  </rdf:Bag>
</plus:Licensor>`),
			wantAlt:      "Default text",
			wantPurchase: "https://example.com/first",
		},
		{
			name: "uses English when x-default is missing and accepts structured attributes",
			packet: xmpPacket(`
<core:AltTextAccessibility>
  <rdf:Alt>
    <rdf:li xml:lang="fr">Français</rdf:li>
    <rdf:li xml:lang="en"> English text </rdf:li>
  </rdf:Alt>
</core:AltTextAccessibility>
<plus:Licensor>
  <rdf:Seq>
    <rdf:li rdf:parseType="Resource" plus:LicensorURL="https://example.com/attribute" />
  </rdf:Seq>
</plus:Licensor>`),
			wantAlt:      "English text",
			wantPurchase: "https://example.com/attribute",
		},
		{
			name: "uses first language and ignores captions",
			packet: xmpPacket(`
<dc:description><rdf:Alt><rdf:li xml:lang="x-default">Not alt text</rdf:li></rdf:Alt></dc:description>
<core:AltTextAccessibility>
  <rdf:Alt>
    <rdf:li xml:lang="ja"> 日本語 </rdf:li>
    <rdf:li xml:lang="fr">Français</rdf:li>
  </rdf:Alt>
</core:AltTextAccessibility>`),
			wantAlt: "日本語",
		},
		{
			name: "accepts compact attribute alt text",
			packet: `<rdf:RDF
  xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
  xmlns:iptc="http://iptc.org/std/Iptc4xmpCore/1.0/xmlns/">
  <rdf:Description iptc:AltTextAccessibility=" Compact alt text " />
</rdf:RDF>`,
			wantAlt: "Compact alt text",
		},
		{
			name:   "missing fields are valid",
			packet: xmpPacket(`<dc:creator><rdf:Seq><rdf:li>Photographer</rdf:li></rdf:Seq></dc:creator>`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := parsePhotoXMP([]byte(test.packet))
			assert.NilError(t, err)
			assert.Equal(t, metadata.Alt, test.wantAlt)
			assert.Equal(t, metadata.PurchaseURL, test.wantPurchase)
		})
	}
}

func TestParsePhotoXMPRejectsMalformedXML(t *testing.T) {
	_, err := parsePhotoXMP([]byte(`<rdf:RDF xmlns:rdf="`))
	assert.ErrorContains(t, err, "cannot parse XMP XML")
}

func TestLoadPhotoXMPFromJPEG(t *testing.T) {
	want := photoXMP{
		Alt:         "Embedded alt text",
		PurchaseURL: "https://example.com/purchase",
	}
	packet := xmpPacket(`
<core:AltTextAccessibility><rdf:Alt><rdf:li xml:lang="x-default">Embedded alt text</rdf:li></rdf:Alt></core:AltTextAccessibility>
<plus:Licensor><rdf:Bag><rdf:li rdf:parseType="Resource"><plus:LicensorURL>https://example.com/purchase</plus:LicensorURL></rdf:li></rdf:Bag></plus:Licensor>`)

	packetWithTerminator := append([]byte(packet), 0)
	metadata, err := loadPhotoXMP(bytes.NewReader(jpegWithXMP(packetWithTerminator)))
	assert.NilError(t, err)
	assert.DeepEqual(t, metadata, want)

	metadata, err = loadPhotoXMP(bytes.NewReader([]byte{0xff, 0xd8, 0xff, 0xd9}))
	assert.NilError(t, err)
	assert.DeepEqual(t, metadata, photoXMP{})
}

func TestApplyPhotoXMP(t *testing.T) {
	im := &img{
		configuredPurchaseURL:    "https://example.com/config",
		hasConfiguredPurchaseURL: true,
	}
	im.applyPhotoXMP(photoXMP{
		Alt:         "Embedded alt",
		PurchaseURL: "https://example.com/embedded",
	})
	assert.Equal(t, im.Alt, "Embedded alt")
	assert.Equal(t, im.PurchaseURL, "https://example.com/config")

	im.hasConfiguredPurchaseURL = false
	im.applyPhotoXMP(photoXMP{})
	assert.Equal(t, im.Alt, "")
	assert.Equal(t, im.PurchaseURL, "")

	im.configuredPurchaseURL = ""
	im.hasConfiguredPurchaseURL = true
	im.applyPhotoXMP(photoXMP{PurchaseURL: "https://example.com/embedded"})
	assert.Equal(t, im.PurchaseURL, "")
}

func xmpPacket(properties string) string {
	return `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF
    xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
    xmlns:core="http://iptc.org/std/Iptc4xmpCore/1.0/xmlns/"
    xmlns:plus="http://ns.useplus.org/ldf/xmp/1.0/"
    xmlns:dc="http://purl.org/dc/elements/1.1/">
    <rdf:Description>` + properties + `</rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`
}

func jpegWithXMP(packet []byte) []byte {
	return addXMPToJPEG([]byte{0xff, 0xd8, 0xff, 0xd9}, packet)
}

func addXMPToJPEG(source, packet []byte) []byte {
	if len(source) < 2 || source[0] != 0xff || source[1] != 0xd8 {
		panic("test source is not a JPEG")
	}
	identifier := []byte("http://ns.adobe.com/xap/1.0/\x00")
	segmentLength := len(identifier) + len(packet) + 2
	if segmentLength > int(^uint16(0)) {
		panic("test XMP packet is too large for a JPEG APP1 segment")
	}

	xmpSegment := []byte{0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(xmpSegment[2:4], uint16(segmentLength))
	xmpSegment = append(xmpSegment, identifier...)
	xmpSegment = append(xmpSegment, packet...)

	// goexif expects the EXIF APP1 segment to precede other APP1 data.
	insertAt := 2
	for offset := 2; offset+4 <= len(source); {
		if source[offset] != 0xff {
			break
		}
		marker := source[offset+1]
		if marker == 0xd9 || marker == 0xda {
			break
		}
		length := int(binary.BigEndian.Uint16(source[offset+2 : offset+4]))
		segmentEnd := offset + 2 + length
		if length < 2 || segmentEnd > len(source) {
			break
		}
		if marker == 0xe1 && offset+10 <= segmentEnd && bytes.Equal(source[offset+4:offset+10], []byte("Exif\x00\x00")) {
			insertAt = segmentEnd
			break
		}
		offset = segmentEnd
	}

	jpeg := make([]byte, 0, len(source)+len(xmpSegment))
	jpeg = append(jpeg, source[:insertAt]...)
	jpeg = append(jpeg, xmpSegment...)
	return append(jpeg, source[insertAt:]...)
}

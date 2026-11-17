package main

import (
	"bytes"
	"encoding/binary"
	"image"
	_ "image/gif" // registers the decoder DecodeConfig reads a header with
	_ "image/png" // registers the decoder DecodeConfig reads a header with
)

// An image is limited in pixels as well as in bytes (D-267, an image is stored and shown
// only when its header puts it within a pixel limit). A file of 5 MB can hold a picture of
// hundreds of megapixels, which a browser decodes into hundreds of megabytes of memory
// and stalls on, and a colleague's machine is asked to show it by the colleague's turn.
const (
	// maxImagePixels is the most pixels an image may have: 40 million, which holds a screenshot
	// of an 8K display and decodes to about 160 MB.
	maxImagePixels = 40_000_000
	// maxImageSide is the longest side an image may have, which is the largest a browser draws.
	maxImageSide = 16384
)

// imageWithinLimits says whether the header of data, a stored kind of image of type typ, puts
// it within the limits. An image whose header cannot be read is not within them: a browser
// that reads it by a rule of its own is not shown to read it as this does not.
func imageWithinLimits(typ string, data []byte) bool {
	w, h, ok := imageSize(typ, data)
	return ok && w > 0 && h > 0 && w <= maxImageSide && h <= maxImageSide && w*h <= maxImagePixels
}

// imageSize reads the width and height of an image from its header, without decoding it.
// The standard library's header reader does that for a PNG, a GIF and a JPEG, stopping at the
// start of the picture's data, and the Go source has it allocate nothing for the picture. It
// has no WebP reader, so a WebP's size is read here.
func imageSize(typ string, data []byte) (w, h int, ok bool) {
	if typ == "image/webp" {
		return webpSize(data)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

// webpSize reads the canvas size of a WebP file from the header of its first chunk, which
// is one of the three forms: lossy (VP8), lossless (VP8L) and extended (VP8X).
func webpSize(b []byte) (w, h int, ok bool) {
	if len(b) < 30 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return 0, 0, false
	}
	switch string(b[12:16]) {
	case "VP8 ":
		if b[23] != 0x9d || b[24] != 0x01 || b[25] != 0x2a {
			return 0, 0, false
		}
		return int(binary.LittleEndian.Uint16(b[26:28]) & 0x3fff), int(binary.LittleEndian.Uint16(b[28:30]) & 0x3fff), true
	case "VP8L":
		if b[20] != 0x2f {
			return 0, 0, false
		}
		bits := binary.LittleEndian.Uint32(b[21:25])
		return int(bits&0x3fff) + 1, int(bits>>14&0x3fff) + 1, true
	case "VP8X":
		return int(b[24]) | int(b[25])<<8 | int(b[26])<<16 + 1, int(b[27]) | int(b[28])<<8 | int(b[29])<<16 + 1, true
	}
	return 0, 0, false
}

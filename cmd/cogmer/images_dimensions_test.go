package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// An image is limited in pixels as well as in bytes (D-267, an image is stored and shown only
// when its header puts it within a pixel limit). The headers below are made by hand, since a
// real file of that many pixels is the thing the limit exists to keep out.

func pngOfSize(w, h int) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(h))
	ihdr[8], ihdr[9] = 8, 2 // 8 bits, truecolor
	binary.Write(&b, binary.BigEndian, uint32(13))
	b.WriteString("IHDR")
	b.Write(ihdr)
	binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(append([]byte("IHDR"), ihdr...)))
	return b.Bytes()
}

func gifOfSize(w, h int) []byte {
	b := []byte("GIF89a")
	b = binary.LittleEndian.AppendUint16(b, uint16(w))
	b = binary.LittleEndian.AppendUint16(b, uint16(h))
	return append(b, 0x00, 0x00, 0x00) // no color table, background, aspect
}

func jpegOfSize(w, h int) []byte {
	b := []byte{0xff, 0xd8, 0xff, 0xc0, 0x00, 0x0b, 0x08}
	b = binary.BigEndian.AppendUint16(b, uint16(h))
	b = binary.BigEndian.AppendUint16(b, uint16(w))
	b = append(b, 0x01, 0x01, 0x11, 0x00)    // one component
	return append(b, 0xff, 0xda, 0x00, 0x02) // the start of the scan, where a header-only read stops
}

func webpHeader(chunk string, payload []byte) []byte {
	b := []byte("RIFF\x00\x00\x00\x00WEBP" + chunk)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(payload)))
	return append(b, payload...)
}

func webpExtendedOfSize(w, h int) []byte {
	p := []byte{0, 0, 0, 0, byte(w - 1), byte((w - 1) >> 8), byte((w - 1) >> 16), byte(h - 1), byte((h - 1) >> 8), byte((h - 1) >> 16)}
	return webpHeader("VP8X", p)
}

func webpLosslessOfSize(w, h int) []byte {
	bits := uint32(w-1) | uint32(h-1)<<14
	p := binary.LittleEndian.AppendUint32([]byte{0x2f}, bits)
	return webpHeader("VP8L", append(p, make([]byte, 8)...))
}

func webpLossyOfSize(w, h int) []byte {
	p := []byte{0, 0, 0, 0x9d, 0x01, 0x2a}
	p = binary.LittleEndian.AppendUint16(p, uint16(w))
	p = binary.LittleEndian.AppendUint16(p, uint16(h))
	return webpHeader("VP8 ", append(p, make([]byte, 8)...))
}

func TestAnImagesSizeIsReadFromItsHeaderForEachKindStored(t *testing.T) {
	var real bytes.Buffer
	_ = jpeg.Encode(&real, image.NewRGBA(image.Rect(0, 0, 30, 20)), nil)
	var gifReal bytes.Buffer
	_ = gif.Encode(&gifReal, image.NewPaletted(image.Rect(0, 0, 12, 9), color.Palette{color.Black, color.White}), nil)
	for name, c := range map[string]struct {
		typ  string
		data []byte
		w, h int
	}{
		"a real png":       {"image/png", pngBytes(t, 1), 8, 8},
		"a real jpeg":      {"image/jpeg", real.Bytes(), 30, 20},
		"a real gif":       {"image/gif", gifReal.Bytes(), 12, 9},
		"a png header":     {"image/png", pngOfSize(5120, 2880), 5120, 2880},
		"a gif header":     {"image/gif", gifOfSize(640, 480), 640, 480},
		"a jpeg header":    {"image/jpeg", jpegOfSize(4000, 3000), 4000, 3000},
		"a lossy webp":     {"image/webp", webpLossyOfSize(1920, 1080), 1920, 1080},
		"a lossless webp":  {"image/webp", webpLosslessOfSize(800, 600), 800, 600},
		"an extended webp": {"image/webp", webpExtendedOfSize(16384, 2), 16384, 2},
	} {
		w, h, ok := imageSize(c.typ, c.data)
		if !ok || w != c.w || h != c.h {
			t.Errorf("%s: read %d by %d (%v), want %d by %d", name, w, h, ok, c.w, c.h)
		}
	}
}

func TestAnImagePastTheLimitsIsNotWithinThem(t *testing.T) {
	for name, c := range map[string]struct {
		typ    string
		data   []byte
		within bool
	}{
		"a screenshot of an 8K display":      {"image/png", pngOfSize(7680, 4320), true},
		"the most pixels":                    {"image/png", pngOfSize(8000, 5000), true},
		"one row past the most pixels":       {"image/png", pngOfSize(8000, 5001), false},
		"a very large png":                   {"image/png", pngOfSize(30000, 30000), false},
		"the longest side":                   {"image/png", pngOfSize(16384, 100), true},
		"a side past the longest":            {"image/png", pngOfSize(16385, 1), false},
		"a very large jpeg":                  {"image/jpeg", jpegOfSize(20000, 20000), false},
		"a very large gif":                   {"image/gif", gifOfSize(60000, 60000), false},
		"a very large lossy webp":            {"image/webp", webpLossyOfSize(16000, 16000), false},
		"a very large extended webp":         {"image/webp", webpExtendedOfSize(100000, 100000), false},
		"a header that cannot be read":       {"image/png", []byte("\x89PNG\r\n\x1a\nnot a header"), false},
		"a zero width":                       {"image/png", pngOfSize(0, 10), false},
		"a webp that is no riff file":        {"image/webp", []byte("RIFX\x00\x00\x00\x00WEBPVP8X\x0a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"), false},
		"a lossy webp with the wrong marker": {"image/webp", append(webpHeader("VP8 ", []byte{0, 0, 0, 1, 2, 3, 10, 0, 10, 0, 0, 0}), make([]byte, 8)...), false},
	} {
		if got := imageWithinLimits(c.typ, c.data); got != c.within {
			t.Errorf("%s: within = %v, want %v", name, got, c.within)
		}
	}
}

func TestAnImageWithTooManyPixelsIsNotSentAndSaysSo(t *testing.T) {
	_, images := sessionFiles(t)
	big := pngOfSize(30000, 30000)
	writeFile(t, filepath.Join(images, "1.png"), big)
	p := prepareImage(pastedFile{N: 1, Path: filepath.Join(images, "1.png")})
	if p.ref.Note != imageTooManyPixels || p.data != nil || p.ref.SHA256 != "" || p.ref.Type != "image/png" {
		t.Errorf("a file: %+v", p.ref)
	}
	q := prepareData(2, big)
	if q.ref.Note != imageTooManyPixels || q.data != nil || q.ref.SHA256 != "" {
		t.Errorf("bytes in memory: %+v", q.ref)
	}
	if ok := prepareData(3, pngBytes(t, 4)); ok.ref.Note != "" || ok.data == nil {
		t.Errorf("a real image was refused: %+v", ok.ref)
	}
}

// A colleague's event is signed, so what it names is the author's word, and the bytes
// are checked against the limits on arrival.
func TestABigPictureAColleagueNamesIsRefusedWhenItArrives(t *testing.T) {
	a, b := imagePair(t)
	big := pngOfSize(30000, 30000)
	ref := a.shareImage(t, big, nil)
	b.d.pullFrom(a.addr)
	b.askOnly(a)
	if n := b.d.fetchWanted(); n != 0 {
		t.Fatalf("a picture past the pixel limit was stored (%d)", n)
	}
	bpath, _ := b.store.imagePath(ref.SHA256)
	for _, p := range []string{bpath, bpath + ".part"} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s is left after a refused image", p)
		}
	}
}

func TestTheViewSaysAnImageHadTooManyPixels(t *testing.T) {
	html := imagesHTML("room", []imageRef{{N: 3, Note: imageTooManyPixels}}, nil)
	if !bytes.Contains([]byte(html), []byte("image 3 has more pixels than cogmer sends")) {
		t.Errorf("the view says %q", html)
	}
}

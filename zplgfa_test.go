package zplgfa

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type zplTest struct {
	Filename    string `json:"filename"`
	Zplstring   string `json:"zplstring"`
	Graphictype string `json:"graphictype"`
}

var zplTests []zplTest

func init() {
	jsonstr, err := os.ReadFile("./tests/tests.json")
	if err != nil {
		log.Fatalf("Failed to read test cases: %s", err)
	}
	if err := json.Unmarshal(jsonstr, &zplTests); err != nil {
		log.Fatalf("Failed to unmarshal test cases: %s", err)
	}
}

func Test_CompressASCII(t *testing.T) {
	if str := CompressASCII("FFFFFFFF000000"); str != "NFL0" {
		t.Fatalf("CompressASCII failed: got %s, want NFL0", str)
	}
}

func Test_ConvertToZPLAt(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 8, 1))

	got := ConvertToZPLAt(img, ASCII, 12, 34)
	want := "^XA,^FS\n^FO12,34\n^GFA,3,1,1,\nFF\n^FS,^XZ\n"

	if got != want {
		t.Fatalf("ConvertToZPLAt failed:\nExpected:\n%s\nGot:\n%s", want, got)
	}
}

func Test_ConvertToZPLWithOptionsReverse(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 8, 1))

	got := ConvertToZPLWithOptions(img, ConvertOptions{
		GraphicType: ASCII,
		X:           5,
		Y:           6,
		Reverse:     true,
	})
	want := "^XA,^FS\n^FO5,6\n^FR\n^GFA,3,1,1,\nFF\n^FS,^XZ\n"

	if got != want {
		t.Fatalf("ConvertToZPLWithOptions failed:\nExpected:\n%s\nGot:\n%s", want, got)
	}
}

func Test_ConvertToZPLSmallImage(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 1, 1))

	got := ConvertToZPL(img, ASCII)
	want := "^XA,^FS\n^FO0,0\n^GFA,3,1,1,\n80\n^FS,^XZ\n"

	if got != want {
		t.Fatalf("ConvertToZPL for small image failed:\nExpected:\n%s\nGot:\n%s", want, got)
	}
}

func Test_ConvertToGraphicFieldZ64(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 8, 1))
	assertZ64GraphicField(t, img, 1, []byte{0xff})

	pattern := image.NewGray(image.Rect(0, 0, 10, 2))
	for y := 0; y < pattern.Bounds().Dy(); y++ {
		for x := 0; x < pattern.Bounds().Dx(); x++ {
			pattern.SetGray(x, y, color.Gray{Y: 0xff})
		}
	}
	for x := 0; x < pattern.Bounds().Dx(); x += 2 {
		pattern.SetGray(x, 0, color.Gray{Y: 0x00})
	}
	assertZ64GraphicField(t, pattern, 2, []byte{0xaa, 0x80, 0x00, 0x00})
}

func assertZ64GraphicField(t *testing.T, img image.Image, bytesPerRow int, wantRaw []byte) {
	t.Helper()
	got := ConvertToGraphicField(img, Z64)
	prefix := fmt.Sprintf("^GFA,%d,%d,%d,\n:Z64:", len(wantRaw), len(wantRaw), bytesPerRow)
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("ConvertToGraphicField Z64 prefix failed:\nExpected prefix:\n%s\nGot:\n%s", prefix, got)
	}

	parts := strings.Split(strings.TrimPrefix(got, prefix), ":")
	if len(parts) != 2 {
		t.Fatalf("ConvertToGraphicField Z64 payload failed: got %q", got)
	}

	compressed, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("ConvertToGraphicField Z64 base64 failed: %s", err)
	}

	if wantCRC := fmt.Sprintf("%04X", crc16CCITT(compressed)); parts[1] != wantCRC {
		t.Fatalf("ConvertToGraphicField Z64 CRC failed: got %s, want %s", parts[1], wantCRC)
	}

	reader, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("ConvertToGraphicField Z64 zlib reader failed: %s", err)
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ConvertToGraphicField Z64 decompress failed: %s", err)
	}

	if !bytes.Equal(raw, wantRaw) {
		t.Fatalf("ConvertToGraphicField Z64 raw data failed: got % X, want % X", raw, wantRaw)
	}
}

func Test_ConvertReaderToZPL(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 1))
	for x := 0; x < img.Bounds().Dx(); x++ {
		img.Set(x, 0, color.NRGBA{A: 0xff})
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("Failed to encode PNG: %s", err)
	}

	got, err := ConvertReaderToZPL(&buf, ASCII)
	if err != nil {
		t.Fatalf("ConvertReaderToZPL failed: %s", err)
	}

	want := "^XA,^FS\n^FO0,0\n^GFA,3,1,1,\nFF\n^FS,^XZ\n"
	if got != want {
		t.Fatalf("ConvertReaderToZPL failed:\nExpected:\n%s\nGot:\n%s", want, got)
	}
}

func Test_ConvertReaderToZPLError(t *testing.T) {
	if _, err := ConvertReaderToZPL(strings.NewReader("not an image"), ASCII); err == nil {
		t.Fatal("ConvertReaderToZPL should fail for invalid image data")
	}
}

func Test_ConvertFileToZPL(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 1))
	for x := 0; x < img.Bounds().Dx(); x++ {
		img.Set(x, 0, color.NRGBA{A: 0xff})
	}

	filename := filepath.Join(t.TempDir(), "label.png")
	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create temp image: %s", err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatalf("Failed to encode temp image: %s", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close temp image: %s", err)
	}

	got, err := ConvertFileToZPL(filename, ASCII)
	if err != nil {
		t.Fatalf("ConvertFileToZPL failed: %s", err)
	}

	want := "^XA,^FS\n^FO0,0\n^GFA,3,1,1,\nFF\n^FS,^XZ\n"
	if got != want {
		t.Fatalf("ConvertFileToZPL failed:\nExpected:\n%s\nGot:\n%s", want, got)
	}
}

func Test_ConvertToZPL(t *testing.T) {
	for i, testcase := range zplTests {
		t.Run(testcase.Filename, func(t *testing.T) {
			testConvertToZPL(t, testcase, i)
		})
	}
}

func testConvertToZPL(t *testing.T, testcase zplTest, index int) {
	file, err := os.Open(testcase.Filename)
	if err != nil {
		t.Fatalf("Failed to open file %s: %s", testcase.Filename, err)
	}
	defer file.Close()

	_, _, err = image.DecodeConfig(file)
	if err != nil {
		t.Fatalf("Failed to decode config for file %s: %s", testcase.Filename, err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("Failed to reset file pointer for %s: %s", testcase.Filename, err)
	}

	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("Failed to decode image for file %s: %s", testcase.Filename, err)
	}

	flat := FlattenImage(img)
	graphicType := parseGraphicType(testcase.Graphictype)
	gfimg := ConvertToZPL(flat, graphicType)

	if graphicType == Binary {
		gfimg = base64.StdEncoding.EncodeToString([]byte(gfimg))
	} else {
		gfimg = cleanZPLString(gfimg)
	}

	if gfimg != testcase.Zplstring {
		t.Fatalf("Testcase %d ConvertToZPL failed for file %s: \nExpected: \n%s\nGot: \n%s\n", index, testcase.Filename, testcase.Zplstring, gfimg)
	}
}

func parseGraphicType(graphicTypeStr string) GraphicType {
	switch graphicTypeStr {
	case "ASCII":
		return ASCII
	case "Binary":
		return Binary
	case "CompressedASCII":
		return CompressedASCII
	case "Z64":
		return Z64
	default:
		return CompressedASCII
	}
}

func cleanZPLString(zpl string) string {
	zpl = strings.ReplaceAll(zpl, " ", "")
	zpl = strings.ReplaceAll(zpl, "\n", "")
	return zpl
}

func Benchmark_CompressASCII(b *testing.B) {
	input := strings.Repeat("F", 512) + strings.Repeat("0", 512) + strings.Repeat("A5", 512)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CompressASCII(input)
	}
}

func Benchmark_ConvertToZPLASCII(b *testing.B) {
	benchmarkConvertToZPL(b, ASCII)
}

func Benchmark_ConvertToZPLCompressedASCII(b *testing.B) {
	benchmarkConvertToZPL(b, CompressedASCII)
}

func Benchmark_ConvertToZPLBinary(b *testing.B) {
	benchmarkConvertToZPL(b, Binary)
}

func benchmarkConvertToZPL(b *testing.B, graphicType GraphicType) {
	img := benchmarkImage(256, 128)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConvertToZPL(img, graphicType)
	}
}

func benchmarkImage(width, height int) image.Image {
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x/8+y/8)%2 == 0 {
				img.SetGray(x, y, color.Gray{Y: 0})
			} else {
				img.SetGray(x, y, color.Gray{Y: 0xff})
			}
		}
	}
	return img
}

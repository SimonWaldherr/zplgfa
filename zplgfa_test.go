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

func Test_ConvertToZPLLines(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 8, 3))
	fillGray(img, color.White)
	for x := 1; x < 4; x++ {
		img.Set(x, 0, color.Black)
	}
	img.Set(6, 0, color.Black)
	for x := 0; x < 8; x++ {
		img.Set(x, 2, color.Black)
	}

	got := ConvertToZPLLinesAt(img, 10, 20)
	want := "^XA,^FS\n" +
		"^FO11,20\n^GB3,1,1^FS\n" +
		"^FO16,20\n^GB1,1,1^FS\n" +
		"^FO10,22\n^GB8,1,1^FS\n" +
		"^XZ\n"
	if got != want {
		t.Fatalf("ConvertToZPLLinesAt failed:\nExpected:\n%s\nGot:\n%s", want, got)
	}
}

func Test_ConvertGraphicFieldToImage(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 8, 2))
	fillGray(img, color.White)
	for x := 0; x < 8; x += 2 {
		img.Set(x, 0, color.Black)
	}
	for x := 1; x < 8; x += 2 {
		img.Set(x, 1, color.Black)
	}

	for _, graphicType := range []GraphicType{ASCII, CompressedASCII, Z64} {
		t.Run(fmt.Sprint(graphicType), func(t *testing.T) {
			zpl := ConvertToZPL(img, graphicType)
			got, err := ConvertZPLToImage(zpl)
			if err != nil {
				t.Fatalf("ConvertZPLToImage failed: %s", err)
			}
			assertGrayImageEqual(t, got, img)
		})
	}
}

func Test_ConvertGraphicFieldToImageCompressedRepeatLine(t *testing.T) {
	got, err := ConvertGraphicFieldToImage("^GFA,5,2,1,\nAA:")
	if err != nil {
		t.Fatalf("ConvertGraphicFieldToImage failed: %s", err)
	}
	want := image.NewGray(image.Rect(0, 0, 8, 2))
	fillGray(want, color.White)
	for y := 0; y < 2; y++ {
		for x := 0; x < 8; x += 2 {
			want.Set(x, y, color.Black)
		}
	}
	assertGrayImageEqual(t, got, want)
}

func Test_ConvertGraphicFieldToImageError(t *testing.T) {
	if _, err := ConvertZPLToImage("^XA^FO0,0^FS^XZ"); err == nil {
		t.Fatal("ConvertZPLToImage should fail without a ^GF field")
	}
	if _, err := ConvertGraphicFieldToImage("^GFA,1,2,1,\nFF"); err == nil {
		t.Fatal("ConvertGraphicFieldToImage should fail for invalid dimensions")
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

func assertZ64GraphicField(t *testing.T, img image.Image, expectedBytesPerRow int, expectedUncompressedData []byte) {
	t.Helper()
	got := ConvertToGraphicField(img, Z64)
	prefix := fmt.Sprintf("^GFA,%d,%d,%d,\n:Z64:", len(expectedUncompressedData), len(expectedUncompressedData), expectedBytesPerRow)
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

	if !bytes.Equal(raw, expectedUncompressedData) {
		t.Fatalf("ConvertToGraphicField Z64 raw data failed: got % X, want % X", raw, expectedUncompressedData)
	}
}

func fillGray(img *image.Gray, c color.Color) {
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

func assertGrayImageEqual(t *testing.T, got, want *image.Gray) {
	t.Helper()
	if !got.Bounds().Eq(want.Bounds()) {
		t.Fatalf("image bounds failed: got %v, want %v", got.Bounds(), want.Bounds())
	}
	for y := want.Bounds().Min.Y; y < want.Bounds().Max.Y; y++ {
		for x := want.Bounds().Min.X; x < want.Bounds().Max.X; x++ {
			if got.GrayAt(x, y) != want.GrayAt(x, y) {
				t.Fatalf("pixel %d,%d failed: got %v, want %v", x, y, got.GrayAt(x, y), want.GrayAt(x, y))
			}
		}
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

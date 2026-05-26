package zplgfa

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

// GraphicType is a type to select the graphic format
type GraphicType int

const (
	// ASCII graphic type using only hex characters (0-9A-F)
	ASCII GraphicType = iota
	// Binary saving the same data as binary
	Binary
	// CompressedASCII compresses the hex data via RLE
	CompressedASCII
	// Z64 compresses the binary data with zlib and encodes it as base64 with a CRC
	Z64
)

// ConvertOptions configures ZPL output created by ConvertToZPLWithOptions.
type ConvertOptions struct {
	GraphicType GraphicType
	X           int
	Y           int
	Reverse     bool
}

// ConvertToZPL wraps ConvertToGraphicField, adding ZPL start and end codes.
func ConvertToZPL(img image.Image, graphicType GraphicType) string {
	return ConvertToZPLWithOptions(img, ConvertOptions{GraphicType: graphicType})
}

// ConvertToZPLAt wraps ConvertToGraphicField, adding ZPL start and end codes and field origin.
func ConvertToZPLAt(img image.Image, graphicType GraphicType, x, y int) string {
	return ConvertToZPLWithOptions(img, ConvertOptions{GraphicType: graphicType, X: x, Y: y})
}

// ConvertToZPLWithOptions wraps ConvertToGraphicField, adding ZPL start and end codes and optional field settings.
func ConvertToZPLWithOptions(img image.Image, options ConvertOptions) string {
	if img == nil || img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		return ""
	}

	reverseField := ""
	if options.Reverse {
		reverseField = "^FR\n"
	}

	return fmt.Sprintf("^XA,^FS\n^FO%d,%d\n%s%s^FS,^XZ\n", options.X, options.Y, reverseField, ConvertToGraphicField(img, options.GraphicType))
}

// ConvertToZPLLines converts black pixel runs to ZPL line/box commands.
func ConvertToZPLLines(img image.Image) string {
	return ConvertToZPLLinesWithOptions(img, ConvertOptions{})
}

// ConvertToZPLLinesAt converts black pixel runs to ZPL line/box commands at the given origin.
func ConvertToZPLLinesAt(img image.Image, x, y int) string {
	return ConvertToZPLLinesWithOptions(img, ConvertOptions{X: x, Y: y})
}

// ConvertToZPLLinesWithOptions converts black pixel runs to ZPL line/box commands.
func ConvertToZPLLinesWithOptions(img image.Image, options ConvertOptions) string {
	if img == nil || img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		return ""
	}

	var zpl strings.Builder
	zpl.WriteString("^XA,^FS\n")
	zpl.WriteString(ConvertToLineFields(img, options))
	zpl.WriteString("^XZ\n")
	return zpl.String()
}

// ConvertToLineFields converts black pixel runs to ZPL ^GB line/box fields.
func ConvertToLineFields(img image.Image, options ConvertOptions) string {
	if img == nil || img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		return ""
	}

	bounds := img.Bounds()
	var fields strings.Builder
	reverseField := ""
	if options.Reverse {
		reverseField = "^FR\n"
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		runStart := -1
		for x := bounds.Min.X; x <= bounds.Max.X; x++ {
			black := false
			if x < bounds.Max.X {
				black = color.Gray16Model.Convert(img.At(x, y)).(color.Gray16).Y < math.MaxUint16/2
			}
			if black && runStart == -1 {
				runStart = x
			}
			if (!black || x == bounds.Max.X) && runStart != -1 {
				fields.WriteString(fmt.Sprintf("^FO%d,%d\n%s^GB%d,1,1^FS\n",
					options.X+runStart-bounds.Min.X,
					options.Y+y-bounds.Min.Y,
					reverseField,
					x-runStart,
				))
				runStart = -1
			}
		}
	}

	return fields.String()
}

// ConvertReaderToZPL decodes PNG, JPEG or GIF image data from reader and converts it to ZPL.
func ConvertReaderToZPL(reader io.Reader, graphicType GraphicType) (string, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return "", err
	}

	return ConvertToZPL(FlattenImage(img), graphicType), nil
}

// ConvertFileToZPL opens an image file, decodes it and converts it to ZPL.
func ConvertFileToZPL(filename string, graphicType GraphicType) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return ConvertReaderToZPL(file, graphicType)
}

// FlattenImage optimizes an image for the converting process.
func FlattenImage(source image.Image) *image.NRGBA {
	size := source.Bounds().Size()
	target := image.NewNRGBA(source.Bounds())
	background := color.White

	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			p := source.At(x, y)
			target.Set(x, y, flatten(p, background))
		}
	}
	return target
}

// flatten blends a pixel with the background color based on its alpha value.
func flatten(input, background color.Color) color.Color {
	src := color.NRGBA64Model.Convert(input).(color.NRGBA64)
	r, g, b, a := src.RGBA()
	bgR, bgG, bgB, _ := background.RGBA()
	alpha := float32(a) / 0xffff

	blend := func(c, bg uint32) uint8 {
		val := 0xffff - uint32(float32(bg)*alpha)
		val |= uint32(float32(c) * alpha)
		return uint8(val >> 8)
	}

	return color.NRGBA{
		R: blend(r, bgR),
		G: blend(g, bgG),
		B: blend(b, bgB),
		A: 0xff,
	}
}

// getRepeatCode generates ZPL repeat codes for character compression.
func getRepeatCode(repeatCount int, char string) string {
	const maxRepeat = 419
	highString := " ghijklmnopqrstuvwxyz"
	lowString := " GHIJKLMNOPQRSTUVWXY"

	encode := func(count int) string {
		var singleRepeatStr strings.Builder
		high := count / 20
		low := count % 20

		if high > 0 {
			singleRepeatStr.WriteByte(highString[high])
		}
		if low > 0 {
			singleRepeatStr.WriteByte(lowString[low])
		}

		singleRepeatStr.WriteString(char)
		return singleRepeatStr.String()
	}

	if repeatCount > maxRepeat {
		var repeatStr strings.Builder
		remainder := repeatCount % maxRepeat
		quotient := repeatCount / maxRepeat

		if remainder > 0 {
			repeatStr.WriteString(encode(remainder))
		}

		maxEncoding := encode(maxRepeat)
		for i := 0; i < quotient; i++ {
			repeatStr.WriteString(maxEncoding)
		}

		return repeatStr.String()
	}

	return encode(repeatCount)
}

// CompressASCII compresses the ASCII data of a ZPL Graphic Field using RLE.
func CompressASCII(input string) string {
	if input == "" {
		return ""
	}

	var output strings.Builder
	var lastChar string
	var lastCharSince int

	for i := 0; i <= len(input); i++ {
		curChar := ""
		if i < len(input) {
			curChar = string(input[i])
		}

		if lastChar != curChar {
			if i-lastCharSince > 4 {
				output.WriteString(getRepeatCode(i-lastCharSince, lastChar))
			} else {
				output.WriteString(strings.Repeat(lastChar, i-lastCharSince))
			}
			lastChar = curChar
			lastCharSince = i
		}

		if curChar == "" && lastCharSince == 0 {
			switch lastChar {
			case "0":
				return ","
			case "F":
				return "!"
			}
		}
	}

	return output.String()
}

// EncodeZ64 compresses binary graphic data and formats it as a Z64 payload.
func EncodeZ64(input []byte) (string, error) {
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(input); err != nil {
		if closeErr := writer.Close(); closeErr != nil {
			return "", fmt.Errorf("zlib write failed: %w (close also failed: %v)", err, closeErr)
		}

		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	compressedBytes := compressed.Bytes()
	return fmt.Sprintf(":Z64:%s:%04X", base64.StdEncoding.EncodeToString(compressedBytes), crc16CCITT(compressedBytes)), nil
}

// ConvertZPLToImage extracts the first ^GF field from a ZPL string and converts it to an image.
func ConvertZPLToImage(zpl string) (*image.Gray, error) {
	start := strings.Index(zpl, "^GF")
	if start == -1 {
		return nil, fmt.Errorf("no ^GF field found")
	}
	return ConvertGraphicFieldToImage(zpl[start:])
}

// ConvertGraphicFieldToImage converts a ZPL ^GF graphic field to a black and white image.
func ConvertGraphicFieldToImage(graphicField string) (*image.Gray, error) {
	gfType, bytesUsed, bytesPerRow, data, err := parseGraphicField(graphicField)
	if err != nil {
		return nil, err
	}
	if bytesPerRow <= 0 || bytesUsed < 0 || bytesUsed%bytesPerRow != 0 {
		return nil, fmt.Errorf("invalid ^GF dimensions")
	}

	var raw []byte
	switch gfType {
	case 'A', 'C':
		raw, err = decodeASCIIData(data, bytesUsed, bytesPerRow)
	case 'B':
		raw, err = decodeBinaryData(data, bytesUsed)
	default:
		err = fmt.Errorf("unsupported ^GF type %q", string(gfType))
	}
	if err != nil {
		return nil, err
	}

	return imageFromGraphicData(raw, bytesPerRow), nil
}

func parseGraphicField(graphicField string) (byte, int, int, string, error) {
	if len(graphicField) < 4 || graphicField[:3] != "^GF" {
		return 0, 0, 0, "", fmt.Errorf("graphic field must start with ^GF")
	}

	gfType := graphicField[3]
	rest := graphicField[4:]
	parts := strings.SplitN(rest, ",", 5)
	if len(parts) != 5 {
		return 0, 0, 0, "", fmt.Errorf("invalid ^GF field")
	}

	bytesUsed, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return 0, 0, 0, "", fmt.Errorf("invalid ^GF byte count: %w", err)
	}
	bytesPerRow, err := strconv.Atoi(strings.TrimSpace(parts[3]))
	if err != nil {
		return 0, 0, 0, "", fmt.Errorf("invalid ^GF bytes per row: %w", err)
	}

	data := parts[4]
	if end := strings.Index(data, "^FS"); end != -1 {
		data = data[:end]
	} else if end := strings.Index(data, "^XZ"); end != -1 {
		data = data[:end]
	}
	data = strings.TrimSpace(data)

	return gfType, bytesUsed, bytesPerRow, data, nil
}

func decodeBinaryData(data string, bytesUsed int) ([]byte, error) {
	raw := []byte(data)
	if len(raw) < bytesUsed {
		return nil, fmt.Errorf("binary ^GF data too short")
	}
	return raw[:bytesUsed], nil
}

func decodeASCIIData(data string, bytesUsed, bytesPerRow int) ([]byte, error) {
	if strings.HasPrefix(data, ":Z64:") {
		return decodeZ64Data(data, bytesUsed)
	}

	rowHexLen := bytesPerRow * 2
	rows, err := expandCompressedASCII(data, rowHexLen, bytesUsed/bytesPerRow)
	if err != nil {
		return nil, err
	}

	raw := make([]byte, 0, bytesUsed)
	for _, row := range rows {
		bytes, err := hex.DecodeString(row)
		if err != nil {
			return nil, err
		}
		raw = append(raw, bytes...)
	}
	if len(raw) != bytesUsed {
		return nil, fmt.Errorf("^GF data size mismatch: got %d bytes, want %d", len(raw), bytesUsed)
	}
	return raw, nil
}

func decodeZ64Data(data string, bytesUsed int) ([]byte, error) {
	parts := strings.Split(data, ":")
	if len(parts) < 4 || parts[1] != "Z64" {
		return nil, fmt.Errorf("invalid Z64 payload")
	}
	compressed, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	if len(parts[3]) >= 4 {
		want := strings.ToUpper(parts[3][:4])
		got := fmt.Sprintf("%04X", crc16CCITT(compressed))
		if want != got {
			return nil, fmt.Errorf("Z64 CRC mismatch: got %s, want %s", got, want)
		}
	}

	reader, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(raw) != bytesUsed {
		return nil, fmt.Errorf("Z64 data size mismatch: got %d bytes, want %d", len(raw), bytesUsed)
	}
	return raw, nil
}

func expandCompressedASCII(data string, rowHexLen, expectedRows int) ([]string, error) {
	highRepeat := " ghijklmnopqrstuvwxyz"
	lowRepeat := " GHIJKLMNOPQRSTUVWXY"
	rows := make([]string, 0, expectedRows)
	var current strings.Builder
	lastRow := ""
	repeat := 0

	appendRow := func(row string) error {
		if len(row) != rowHexLen {
			return fmt.Errorf("^GF row length mismatch: got %d nibbles, want %d", len(row), rowHexLen)
		}
		rows = append(rows, row)
		lastRow = row
		current.Reset()
		return nil
	}

	for _, r := range data {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			continue
		}
		if r == ':' {
			if current.Len() != 0 {
				return nil, fmt.Errorf("repeat-line marker inside a row")
			}
			if lastRow == "" {
				return nil, fmt.Errorf("repeat-line marker without previous row")
			}
			rows = append(rows, lastRow)
			continue
		}
		if r == ',' || r == '!' {
			if current.Len() != 0 {
				return nil, fmt.Errorf("full-line marker inside a row")
			}
			fill := "0"
			if r == '!' {
				fill = "F"
			}
			if err := appendRow(strings.Repeat(fill, rowHexLen)); err != nil {
				return nil, err
			}
			continue
		}
		if idx := strings.IndexRune(highRepeat, r); idx > 0 {
			repeat += idx * 20
			continue
		}
		if idx := strings.IndexRune(lowRepeat, r); idx > 0 {
			repeat += idx
			continue
		}
		if !isHexRune(r) {
			return nil, fmt.Errorf("invalid ^GF data character %q", r)
		}

		count := repeat
		if count == 0 {
			count = 1
		}
		current.WriteString(strings.Repeat(strings.ToUpper(string(r)), count))
		repeat = 0
		for current.Len() >= rowHexLen {
			row := current.String()[:rowHexLen]
			remaining := current.String()[rowHexLen:]
			if err := appendRow(row); err != nil {
				return nil, err
			}
			current.WriteString(remaining)
		}
	}

	if current.Len() != 0 {
		if err := appendRow(current.String()); err != nil {
			return nil, err
		}
	}
	if repeat != 0 {
		return nil, fmt.Errorf("dangling repeat count in ^GF data")
	}
	if len(rows) != expectedRows {
		return nil, fmt.Errorf("^GF row count mismatch: got %d rows, want %d", len(rows), expectedRows)
	}
	return rows, nil
}

func isHexRune(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'A' && r <= 'F') || (r >= 'a' && r <= 'f')
}

func imageFromGraphicData(raw []byte, bytesPerRow int) *image.Gray {
	height := len(raw) / bytesPerRow
	width := bytesPerRow * 8
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := color.White
			if raw[y*bytesPerRow+x/8]&(1<<(7-uint(x)%8)) != 0 {
				pixel = color.Black
			}
			img.Set(x, y, pixel)
		}
	}
	return img
}

func crc16CCITT(data []byte) uint16 {
	var crc uint16 = 0xffff
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// ConvertToGraphicField converts an image.Image to a ZPL compatible Graphic Field.
// For compatibility with the original string-only API, Z64 encoding errors are represented as an empty string.
// Use ConvertToGraphicFieldWithError when callers need to distinguish encoding errors from empty output.
func ConvertToGraphicField(source image.Image, graphicType GraphicType) string {
	graphicField, err := ConvertToGraphicFieldWithError(source, graphicType)
	if err != nil {
		return ""
	}
	return graphicField
}

// ConvertToGraphicFieldWithError converts an image.Image to a ZPL compatible Graphic Field and returns encoding errors.
func ConvertToGraphicFieldWithError(source image.Image, graphicType GraphicType) (string, error) {
	var gfType, graphicFieldData string
	size := source.Bounds().Size()
	width := (size.X + 7) / 8 // round up division
	height := size.Y
	var lastLine string
	var rawGraphicData []byte
	if graphicType == Z64 {
		rawGraphicData = make([]byte, 0, width*height)
	}

	for y := 0; y < height; y++ {
		line := make([]uint8, width)
		for x := 0; x < size.X; x++ {
			if x%8 == 0 {
				line[x/8] = 0
			}
			if lum := color.Gray16Model.Convert(source.At(x, y)).(color.Gray16).Y; lum < math.MaxUint16/2 {
				line[x/8] |= 1 << (7 - uint(x)%8)
			}
		}

		if graphicType == Z64 {
			rawGraphicData = append(rawGraphicData, line...)
			continue
		}

		hexStr := strings.ToUpper(hex.EncodeToString(line))
		switch graphicType {
		case ASCII:
			graphicFieldData += fmt.Sprintln(hexStr)
		case CompressedASCII:
			curLine := CompressASCII(hexStr)
			if lastLine == curLine {
				graphicFieldData += ":"
			} else {
				graphicFieldData += curLine
			}
			lastLine = curLine
		case Binary:
			graphicFieldData += string(line)
		}
	}

	totalBytes := len(graphicFieldData)
	switch graphicType {
	case ASCII, CompressedASCII:
		gfType = "A"
	case Binary:
		gfType = "B"
	case Z64:
		gfType = "A"
		totalBytes = len(rawGraphicData)
		encoded, err := EncodeZ64(rawGraphicData)
		if err != nil {
			return "", err
		}
		graphicFieldData = encoded
	}

	return fmt.Sprintf("^GF%s,%d,%d,%d,\n%s", gfType, totalBytes, width*height, width, graphicFieldData), nil
}

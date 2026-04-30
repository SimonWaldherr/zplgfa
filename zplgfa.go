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
func EncodeZ64(input []byte) string {
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	// The zlib writer is backed by bytes.Buffer, whose Write method cannot fail.
	_, _ = writer.Write(input)
	_ = writer.Close()

	compressedBytes := compressed.Bytes()
	return fmt.Sprintf(":Z64:%s:%04X", base64.StdEncoding.EncodeToString(compressedBytes), crc16CCITT(compressedBytes))
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
func ConvertToGraphicField(source image.Image, graphicType GraphicType) string {
	var gfType, graphicFieldData string
	size := source.Bounds().Size()
	width := (size.X + 7) / 8 // round up division
	height := size.Y
	var lastLine string
	rawGraphicData := make([]byte, 0, width*height)

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

		rawGraphicData = append(rawGraphicData, line...)
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
		graphicFieldData = EncodeZ64(rawGraphicData)
	}

	return fmt.Sprintf("^GF%s,%d,%d,%d,\n%s", gfType, totalBytes, width*height, width, graphicFieldData)
}

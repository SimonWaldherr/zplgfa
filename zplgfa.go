package zplgfa

import (
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"math"
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
)

// ConvertToZPL is a wrapper for ConvertToGraphicField, adding ZPL start and end codes.
func ConvertToZPL(img image.Image, graphicType GraphicType) string {
	if img.Bounds().Size().X/8 == 0 {
		return ""
	}
	return fmt.Sprintf("^XA,^FS\n^FO0,0\n%s^FS,^XZ\n", ConvertToGraphicField(img, graphicType))
}

// FlattenImage optimizes an image for the converting process.
func FlattenImage(source image.Image) *image.NRGBA {
	size := source.Bounds().Size()
	background := color.White
	target := image.NewNRGBA(source.Bounds())
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			p := source.At(x, y)
			target.Set(x, y, flatten(p, background))
		}
	}
	return target
}

func flatten(input, background color.Color) color.Color {
	source := color.NRGBA64Model.Convert(input).(color.NRGBA64)
	r, g, b, a := source.RGBA()
	bgR, bgG, bgB, _ := background.RGBA()
	alpha := float32(a) / 0xffff

	conv := func(c, bg uint32) uint8 {
		val := 0xffff - uint32(float32(bg)*alpha)
		val |= uint32(float32(c) * alpha)
		return uint8(val >> 8)
	}

	return color.NRGBA{
		R: conv(r, bgR),
		G: conv(g, bgG),
		B: conv(b, bgB),
		A: 0xff,
	}
}

func getRepeatCode(repeatCount int, char string) string {
	repeatStr := ""
	if repeatCount > 419 {
		repeatStr += getRepeatCode(repeatCount-419, char)
		repeatCount = 419
	}

	high := repeatCount / 20
	low := repeatCount % 20

	lowString := " GHIJKLMNOPQRSTUVWXY"
	highString := " ghijklmnopqrstuvwxyz"

	if high > 0 {
		repeatStr += string(highString[high])
	}
	if low > 0 {
		repeatStr += string(lowString[low])
	}

	return repeatStr + char
}

// CompressASCII compresses the ASCII data of a ZPL Graphic Field using RLE.
func CompressASCII(input string) string {
	var output, lastChar, repCode string
	lastCharSince := 0

	for i := 0; i < len(input)+1; i++ {
		curChar := ""
		if i < len(input) {
			curChar = string(input[i])
		}

		if lastChar != curChar {
			if i-lastCharSince > 4 {
				repCode = getRepeatCode(i-lastCharSince, lastChar)
				output += repCode
			} else {
				output += strings.Repeat(lastChar, i-lastCharSince)
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

	if output == "" {
		output += getRepeatCode(len(input), lastChar)
	}

	return output
}

// ConvertToGraphicField converts an image.Image to a ZPL compatible Graphic Field.
func ConvertToGraphicField(source image.Image, graphicType GraphicType) string {
	var gfType, lastLine, graphicFieldData string
	size := source.Bounds().Size()
	width := (size.X + 7) / 8 // round up division
	height := size.Y

	for y := 0; y < size.Y; y++ {
		line := make([]uint8, width)
		for x := 0; x < size.X; x++ {
			if x%8 == 0 {
				line[x/8] = 0
			}
			if lum := color.Gray16Model.Convert(source.At(x, y)).(color.Gray16).Y; lum < math.MaxUint16/2 {
				line[x/8] |= 1 << (7 - uint(x)%8)
			}
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

	switch graphicType {
	case ASCII, CompressedASCII:
		gfType = "A"
	case Binary:
		gfType = "B"
	}

	return fmt.Sprintf("^GF%s,%d,%d,%d,\n%s", gfType, len(graphicFieldData), width*height, width, graphicFieldData)
}

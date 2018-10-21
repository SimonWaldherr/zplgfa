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

// ConvertToZPL is just a wrapper for ConvertToGraphicField which also includes the ZPL
// starting code ^XA and ending code ^XZ, as well as a Field Separator and Field Origin.
func ConvertToZPL(img image.Image, graphicType GraphicType) string {
	return fmt.Sprintf("^XA,^FS\n^FO0,0\n%s^FS,^XZ\n", ConvertToGraphicField(img, graphicType))
}

// FlattenImage optimizes an image for the converting process
func FlattenImage(source image.Image) *image.NRGBA {
	size := source.Bounds().Size()
	background := color.White
	target := image.NewNRGBA(source.Bounds())
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			p := source.At(x, y)
			flat := flatten(p, background)
			target.Set(x, y, flat)
		}
	}
	return target
}

func flatten(input color.Color, background color.Color) color.Color {
	source := color.NRGBA64Model.Convert(input).(color.NRGBA64)
	r, g, b, a := source.RGBA()
	bgR, bgG, bgB, _ := background.RGBA()
	alpha := float32(a) / 0xffff
	conv := func(c uint32, bg uint32) uint8 {
		val := 0xffff - uint32((float32(bg) * alpha))
		val = val | uint32(float32(c)*alpha)
		return uint8(val >> 8)
	}
	c := color.NRGBA{
		conv(r, bgR),
		conv(g, bgG),
		conv(b, bgB),
		uint8(0xff),
	}
	return c
}

func getRepeatCode(repeatCount int, char string) string {
	repeatStr := ""
	if repeatCount > 419 {
		repeatCount -= 419
		repeatStr += getRepeatCode(repeatCount, char)
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

	repeatStr += char

	return repeatStr
}

// CompressASCII compresses the ASCII data of a ZPL Graphic Field using RLE
func CompressASCII(in string) string {
	var curChar string
	var lastChar string
	var lastCharSince int
	var output string
	var repCode string

	for i := 0; i < len(in)+1; i++ {
		if i == len(in) {
			curChar = ""
			if lastCharSince == 0 {
				switch lastChar {
				case "0":
					output = ","
					return output
				case "F":
					output = "!"
					return output
				}
			}
		} else {
			curChar = string(in[i])
		}
		if lastChar != curChar {
			if i-lastCharSince > 4 {
				repCode = getRepeatCode(i-lastCharSince, lastChar)
				output += repCode
			} else {
				for j := 0; j < i-lastCharSince; j++ {
					output += lastChar
				}
			}

			lastChar = curChar
			lastCharSince = i
		}
	}

	if output == "" {
		output += getRepeatCode(len(in), lastChar)
	}

	return output
}

// ConvertToGraphicField converts an image.Image picture to a ZPL compatible Graphic Field.
// The ZPL ^GF (Graphic Field) supports various data formats, this package supports the
// normal ASCII encoded, as well as a RLE compressed ASCII format. It also supports the
// Binary Graphic Field format. The encoding can be chosen by the second argument.
func ConvertToGraphicField(source image.Image, graphicType GraphicType) string {
	var gfType string
	var lastLine string
	size := source.Bounds().Size()
	width := size.X / 8
	height := size.Y
	if size.Y%8 != 0 {
		width = width + 1
	}

	var GraphicFieldData string

	for y := 0; y < size.Y; y++ {
		line := make([]uint8, width)
		lineIndex := 0
		index := uint8(0)
		currentByte := line[lineIndex]
		for x := 0; x < size.X; x++ {
			index = index + 1
			p := source.At(x, y)
			lum := color.Gray16Model.Convert(p).(color.Gray16)
			if lum.Y < math.MaxUint16/2 {
				currentByte = currentByte | (1 << (8 - index))
			}
			if index >= 8 {
				line[lineIndex] = currentByte
				lineIndex++
				if lineIndex < len(line) {
					currentByte = line[lineIndex]
				}
				index = 0
			}
		}

		hexstr := strings.ToUpper(hex.EncodeToString(line))

		switch graphicType {
		case ASCII:
			GraphicFieldData += fmt.Sprintln(hexstr)
		case CompressedASCII:
			curLine := CompressASCII(hexstr)
			if lastLine == curLine {
				GraphicFieldData += ":"
			} else {
				GraphicFieldData += curLine
			}
			lastLine = curLine
		case Binary:
			GraphicFieldData += fmt.Sprintf("%s", line)
		}
	}

	if graphicType == ASCII || graphicType == CompressedASCII {
		gfType = "A"
	} else if graphicType == Binary {
		gfType = "B"
	}

	return fmt.Sprintf("^GF%s,%d,%d,%d,\n%s", gfType, len(GraphicFieldData), width*height, width, GraphicFieldData)
}

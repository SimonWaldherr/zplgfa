package main

import (
	"flag"
	"fmt"
	"github.com/anthonynsimon/bild/blur"
	"github.com/anthonynsimon/bild/effect"
	"github.com/anthonynsimon/bild/segment"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"
	"simonwaldherr.de/go/zplgfa"
	"strings"
)

func main() {
	var filenameFlag string
	var graphicTypeFlag string
	var imageEditFlag string
	var imageResizeFlag float64
	var graphicType zplgfa.GraphicType

	flag.StringVar(&filenameFlag, "file", "", "filename to convert to zpl")
	flag.StringVar(&graphicTypeFlag, "type", "CompressedASCII", "type of graphic field encoding")
	flag.StringVar(&imageEditFlag, "edit", "", "manipulate the image [invert,monochrome]")
	flag.Float64Var(&imageResizeFlag, "resize", 1.0, "zoom/resize the image")

	// load flag input arguments
	flag.Parse()

	// open file
	file, err := os.Open(filenameFlag)
	if err != nil {
		log.Printf("Warning: could not open the file \"%s\": %s\n", filenameFlag, err)
		return
	}

	defer file.Close()

	// load image head information
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		log.Printf("Warning: image not compatible, format: %s, config: %v, error: %s\n", format, config, err)
	}

	// reset file pointer to the beginning of the file
	file.Seek(0, 0)

	// load and decode image
	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("Warning: could not decode the file, %s\n", err)
		return
	}

	// select graphic field type
	switch graphicTypeFlag {
	case "ASCII":
		graphicType = zplgfa.ASCII
	case "Binary":
		graphicType = zplgfa.Binary
	case "CompressedASCII":
		graphicType = zplgfa.CompressedASCII
	default:
		graphicType = zplgfa.CompressedASCII
	}

	// apply image manipulation functions
	if strings.Contains(imageEditFlag, "monochrome") {
		img = editImageMonochrome(img)
	}
	if strings.Contains(imageEditFlag, "blur") {
		img = blur.Gaussian(img, float64(config.Width)/300)
	}
	if strings.Contains(imageEditFlag, "edge") {
		img = effect.Sobel(img)
	}
	if strings.Contains(imageEditFlag, "segment") {
		img = segment.Threshold(img, 128)
	}
	if strings.Contains(imageEditFlag, "invert") {
		img = editImageInvert(img)
	}

	// resize image
	if imageResizeFlag != 1.0 {
		img = resize.Resize(uint(float64(config.Width)*imageResizeFlag), uint(float64(config.Height)*imageResizeFlag), img, resize.MitchellNetravali)
	}

	// flatten image
	flat := zplgfa.FlattenImage(img)

	// convert image to zpl compatible type
	gfimg := zplgfa.ConvertToZPL(flat, graphicType)

	// output zpl with graphic field date to stdout
	fmt.Println(gfimg)
}

type imageSet interface {
	Set(x, y int, c color.Color)
}

func editImageInvert(img image.Image) image.Image {
	b := img.Bounds()

	imgSet := img.(imageSet)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			oldPixel := img.At(x, y)
			r, g, b, a := oldPixel.RGBA()
			r = 65535 - r
			g = 65535 - g
			b = 65535 - b
			pixel := color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
			imgSet.Set(x, y, pixel)
		}
	}
	return img
}

func editImageMonochrome(img image.Image) image.Image {
	b := img.Bounds()

	imgSet := img.(imageSet)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			oldPixel := img.At(x, y)
			r, g, b, a := oldPixel.RGBA()
			if r > math.MaxUint16/2 || g > math.MaxUint16/2 || b > math.MaxUint16/2 {
				r, g, b = 65535, 65535, 65535
			} else {
				r, g, b = 0, 0, 0
			}
			pixel := color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
			imgSet.Set(x, y, pixel)
		}
	}
	return img
}

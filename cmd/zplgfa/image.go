package main

import (
	"image"
	"image/color"
	"math"
)

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

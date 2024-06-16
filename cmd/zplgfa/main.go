package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"strings"

	"github.com/anthonynsimon/bild/blur"
	"github.com/anthonynsimon/bild/effect"
	"github.com/anthonynsimon/bild/segment"
	"github.com/nfnt/resize"

	"simonwaldherr.de/go/zplgfa"
)

func handleZebraCommands(cmd, ip, port string) bool {
	if ip == "" {
		return false
	}

	cmdActions := map[string]func(string, string) (string, error){
		"cancel": sendCancelCmdToZebra,
		"calib":  sendCalibCmdToZebra,
		"feed":   sendFeedCmdToZebra,
		"info":   getInfoFromZebra,
		"config": getConfigFromZebra,
		"diag":   getDiagFromZebra,
	}

	for key, action := range cmdActions {
		if strings.Contains(cmd, key) {
			result, err := action(ip, port)
			if err == nil {
				if result != "" {
					fmt.Println(result)
				}
				return true
			}
		}
	}
	return false
}

func parseFlags() (string, string, string, string, string, string, float64) {
	var filename, zebraCmd, graphicType, imageEdit, ip, port string
	var resizeFactor float64

	flag.StringVar(&filename, "file", "", "filename to convert to zpl")
	flag.StringVar(&zebraCmd, "cmd", "", "send special command to printer [cancel,calib,feed,info,config,diag]")
	flag.StringVar(&graphicType, "type", "CompressedASCII", "type of graphic field encoding")
	flag.StringVar(&imageEdit, "edit", "", "manipulate the image [invert,monochrome]")
	flag.StringVar(&ip, "ip", "", "send zpl to printer")
	flag.StringVar(&port, "port", "9100", "network port of printer")
	flag.Float64Var(&resizeFactor, "resize", 1.0, "zoom/resize the image")

	flag.Parse()
	return filename, zebraCmd, graphicType, imageEdit, ip, port, resizeFactor
}

func openImageFile(filename string) (image.Image, image.Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, image.Config{}, fmt.Errorf("could not open the file \"%s\": %s", filename, err)
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, config, fmt.Errorf("image not compatible, format: %s, config: %v, error: %s", format, config, err)
	}

	file.Seek(0, 0)
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, config, fmt.Errorf("could not decode the file, %s", err)
	}

	return img, config, nil
}

func processImage(img image.Image, editFlag string, resizeFactor float64, config image.Config) image.Image {
	if strings.Contains(editFlag, "monochrome") {
		img = editImageMonochrome(img)
	}
	if strings.Contains(editFlag, "blur") {
		img = blur.Gaussian(img, float64(config.Width)/300)
	}
	if strings.Contains(editFlag, "edge") {
		img = effect.Sobel(img)
	}
	if strings.Contains(editFlag, "segment") {
		img = segment.Threshold(img, 128)
	}
	if strings.Contains(editFlag, "invert") {
		img = editImageInvert(img)
	}

	if resizeFactor != 1.0 {
		img = resize.Resize(uint(float64(config.Width)*resizeFactor), uint(float64(config.Height)*resizeFactor), img, resize.MitchellNetravali)
	}

	return img
}

func getGraphicType(typeFlag string) zplgfa.GraphicType {
	switch strings.ToUpper(typeFlag) {
	case "ASCII":
		return zplgfa.ASCII
	case "BINARY":
		return zplgfa.Binary
	case "COMPRESSEDASCII":
		return zplgfa.CompressedASCII
	default:
		return zplgfa.CompressedASCII
	}
}

func main() {
	filename, zebraCmd, graphicTypeFlag, imageEdit, ip, port, resizeFactor := parseFlags()

	if handleZebraCommands(zebraCmd, ip, port) && filename == "" {
		return
	}

	if filename == "" {
		log.Printf("Warning: no input file specified\n")
		return
	}

	img, config, err := openImageFile(filename)
	if err != nil {
		log.Printf("Warning: %s\n", err)
		return
	}

	img = processImage(img, imageEdit, resizeFactor, config)

	flat := zplgfa.FlattenImage(img)
	gfimg := zplgfa.ConvertToZPL(flat, getGraphicType(graphicTypeFlag))

	if ip != "" {
		sendDataToZebra(ip, port, gfimg)
	} else {
		fmt.Println(gfimg)
	}
}

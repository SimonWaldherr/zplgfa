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

func specialCmds(zebraCmdFlag, networkIpFlag, networkPortFlag string) bool {
	var cmdSent bool
	if networkIpFlag == "" {
		return cmdSent
	}
	if strings.Contains(zebraCmdFlag, "cancel") {
		if err := sendCancelCmdToZebra(networkIpFlag, networkPortFlag); err == nil {
			cmdSent = true
		}
	}
	if strings.Contains(zebraCmdFlag, "calib") {
		if err := sendCalibCmdToZebra(networkIpFlag, networkPortFlag); err == nil {
			cmdSent = true
		}
	}
	if strings.Contains(zebraCmdFlag, "feed") {
		if err := sendFeedCmdToZebra(networkIpFlag, networkPortFlag); err == nil {
			cmdSent = true
		}
	}
	if strings.Contains(zebraCmdFlag, "info") {
		info, err := getInfoFromZebra(networkIpFlag, networkPortFlag)
		if err == nil {
			fmt.Println(info)
			cmdSent = true
		}
	}
	if strings.Contains(zebraCmdFlag, "config") {
		info, err := getConfigFromZebra(networkIpFlag, networkPortFlag)
		if err == nil {
			fmt.Println(info)
			cmdSent = true
		}
	}
	if strings.Contains(zebraCmdFlag, "diag") {
		info, err := getDiagFromZebra(networkIpFlag, networkPortFlag)
		if err == nil {
			fmt.Println(info)
			cmdSent = true
		}
	}
	return cmdSent
}

func main() {
	var filenameFlag string
	var zebraCmdFlag string
	var graphicTypeFlag string
	var imageEditFlag string
	var networkIpFlag string
	var networkPortFlag string
	var imageResizeFlag float64
	var graphicType zplgfa.GraphicType

	flag.StringVar(&filenameFlag, "file", "", "filename to convert to zpl")
	flag.StringVar(&zebraCmdFlag, "cmd", "", "send special command to printer [cancel,calib,feed,info,config,diag]")
	flag.StringVar(&graphicTypeFlag, "type", "CompressedASCII", "type of graphic field encoding")
	flag.StringVar(&imageEditFlag, "edit", "", "manipulate the image [invert,monochrome]")
	flag.StringVar(&networkIpFlag, "ip", "", "send zpl to printer")
	flag.StringVar(&networkPortFlag, "port", "9100", "network port of printer")
	flag.Float64Var(&imageResizeFlag, "resize", 1.0, "zoom/resize the image")

	// load flag input arguments
	flag.Parse()

	// send special commands to printer
	cmdSent := specialCmds(zebraCmdFlag, networkIpFlag, networkPortFlag)

	// check input parameter
	if filenameFlag == "" {
		if cmdSent {
			return
		}
		log.Printf("Warning: no input file specified\n")
		return
	}

	// open file
	file, err := os.Open(filenameFlag)
	if err != nil {
		log.Printf("Warning: could not open the file \"%s\": %s\n", filenameFlag, err)
		return
	}

	// close file when complete
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
	switch strings.ToUpper(graphicTypeFlag) {
	case "ASCII":
		graphicType = zplgfa.ASCII
	case "BINARY":
		graphicType = zplgfa.Binary
	case "COMPRESSEDASCII":
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

	if networkIpFlag != "" {
		// send zpl to printer
		sendDataToZebra(networkIpFlag, networkPortFlag, gfimg)
	} else {
		// output zpl with graphic field data to stdout
		fmt.Println(gfimg)
	}
}

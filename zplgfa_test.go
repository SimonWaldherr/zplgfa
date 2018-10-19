package zplgfa

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"testing"
	"strings"
)

type zplTest struct {
	file string
	zpl string
}

var zplTests []zplTest

func init() {
	zplTests = []zplTest{
		{
			file: "./test.png",
			zpl: "^XA,^FS^FO0,0^GFA,32,51,3,,::01C000::,001C00::,1DDC00::,::^FS,^XZ",
		},
		{
			file: "./test2.png",
			zpl: "^XA,^FS^FO0,0^GFA,389,630,63,,038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038038000::1C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C71C7100::E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00E00:lJF00^FS,^XZ",
		},
	}
}

func Test_CompressASCII(t *testing.T) {
	if str := CompressASCII("FFFFFFFF000000"); str != "NFL0" {
		t.Fatalf("CompressASCII failed")
	}
}



func Test_ConvertToZPL(t *testing.T) {
	for i, testcase := range zplTests {
		
		filename, zplstring := testcase.file, testcase.zpl
		// open file
		file, err := os.Open(filename)
		if err != nil {
			log.Printf("Warning: could not open the file: %s\n", err)
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
		
		// flatten image
		flat := FlattenImage(img)
		
		// convert image to zpl compatible type
		gfimg := ConvertToZPL(flat, CompressedASCII)
		
		// remove whitespace - only for the test
		gfimg = strings.Replace(gfimg, " ", "", -1)
		gfimg = strings.Replace(gfimg, "\n", "", -1)
		
		if gfimg != zplstring {
			t.Fatalf("Testcase %d ConvertToZPL failed", i)
		}
	}
}

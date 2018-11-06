package zplgfa

import (
	"encoding/base64"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"os"
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
	jsonstr, _ := ioutil.ReadFile("./tests/tests.json")
	json.Unmarshal(jsonstr, &zplTests)
}

func Test_CompressASCII(t *testing.T) {
	if str := CompressASCII("FFFFFFFF000000"); str != "NFL0" {
		t.Fatalf("CompressASCII failed")
	}
}

func Test_ConvertToZPL(t *testing.T) {
	var graphicType GraphicType
	for i, testcase := range zplTests {
		filename, zplstring, graphictype := testcase.Filename, testcase.Zplstring, testcase.Graphictype
		// open file
		file, err := os.Open(filename)
		if err != nil {
			log.Printf("Warning: could not open the file \"%s\": %s\n", filename, err)
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
		switch graphictype {
		case "ASCII":
			graphicType = ASCII
		case "Binary":
			graphicType = Binary
		case "CompressedASCII":
			graphicType = CompressedASCII
		default:
			graphicType = CompressedASCII
		}

		gfimg := ConvertToZPL(flat, graphicType)

		if graphictype == "Binary" {
			gfimg = base64.StdEncoding.EncodeToString([]byte(gfimg))
		} else {
			// remove whitespace - only for the test
			gfimg = strings.Replace(gfimg, " ", "", -1)
			gfimg = strings.Replace(gfimg, "\n", "", -1)
		}

		if gfimg != zplstring {
			log.Printf("ConvertToZPL Test for file \"%s\" failed, wanted: \n%s\ngot: \n%s\n", filename, zplstring, gfimg)
			t.Fatalf("Testcase %d ConvertToZPL failed", i)
		}
	}
}

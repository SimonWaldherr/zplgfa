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
	jsonstr, err := ioutil.ReadFile("./tests/tests.json")
	if err != nil {
		log.Fatalf("Failed to read test cases: %s", err)
	}
	if err := json.Unmarshal(jsonstr, &zplTests); err != nil {
		log.Fatalf("Failed to unmarshal test cases: %s", err)
	}
}

func Test_CompressASCII(t *testing.T) {
	if str := CompressASCII("FFFFFFFF000000"); str != "NFL0" {
		t.Fatalf("CompressASCII failed: got %s, want NFL0", str)
	}
}

func Test_ConvertToZPL(t *testing.T) {
	for i, testcase := range zplTests {
		t.Run(testcase.Filename, func(t *testing.T) {
			testConvertToZPL(t, testcase, i)
		})
	}
}

func testConvertToZPL(t *testing.T, testcase zplTest, index int) {
	file, err := os.Open(testcase.Filename)
	if err != nil {
		t.Fatalf("Failed to open file %s: %s", testcase.Filename, err)
	}
	defer file.Close()

	_, _, err = image.DecodeConfig(file)
	if err != nil {
		t.Fatalf("Failed to decode config for file %s: %s", testcase.Filename, err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("Failed to reset file pointer for %s: %s", testcase.Filename, err)
	}

	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("Failed to decode image for file %s: %s", testcase.Filename, err)
	}

	flat := FlattenImage(img)
	graphicType := parseGraphicType(testcase.Graphictype)
	gfimg := ConvertToZPL(flat, graphicType)

	if graphicType == Binary {
		gfimg = base64.StdEncoding.EncodeToString([]byte(gfimg))
	} else {
		gfimg = cleanZPLString(gfimg)
	}

	if gfimg != testcase.Zplstring {
		t.Fatalf("Testcase %d ConvertToZPL failed for file %s: \nExpected: \n%s\nGot: \n%s\n", index, testcase.Filename, testcase.Zplstring, gfimg)
	}
}

func parseGraphicType(graphicTypeStr string) GraphicType {
	switch graphicTypeStr {
	case "ASCII":
		return ASCII
	case "Binary":
		return Binary
	case "CompressedASCII":
		return CompressedASCII
	default:
		return CompressedASCII
	}
}

func cleanZPLString(zpl string) string {
	zpl = strings.ReplaceAll(zpl, " ", "")
	zpl = strings.ReplaceAll(zpl, "\n", "")
	return zpl
}

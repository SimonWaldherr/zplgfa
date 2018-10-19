package zplgfa

import (
	"testing"
)

func Test_CompressASCII(t *testing.T) {
	if str := CompressASCII("FFFFFFFF000000"); str != "NFL0" {
		t.Fatalf("CompressASCII failed")
	}
}

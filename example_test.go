package zplgfa_test

import (
	"fmt"
	"simonwaldherr.de/go/zplgfa"
)

func ExampleCompressASCII() {
	str := zplgfa.CompressASCII("FFFFFFFF000000")
	fmt.Print(str)

	// Output: NFL0
}

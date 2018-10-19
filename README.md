# ZPLGFA Golang Package

The ZPLGFA Golang package implements some functions to convert PNG, JPEG and GIF files to ZPL compatible ^GF-elements ([Graphic Fields](https://www.zebra.com/us/en/support-downloads/knowledge-articles/gf-graphic-field-zpl-command.html)).

## install

1. [install Golang](https://golang.org/doc/install)
1. `go get simonwaldherr.de/go/zplgfa`

## example

```go
package main

import (
    "simonwaldherr.de/go/zplgfa"
    "fmt"
    "image"
    _ "image/gif"
    _ "image/jpeg"
    _ "image/png"
    "log"
    "os"
)

func main() {
    // open file
    file, err := os.Open("label.png")
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
    flat := zplgfa.FlattenImage(img)

    // convert image to zpl compatible type
    gfimg := zplgfa.ConvertToZPL(flat, zplgfa.CompressedASCII)

    // output zpl with graphic field date to stdout
    fmt.Println(gfimg)
}

```
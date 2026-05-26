# ZPLGFA Golang Package

*convert pictures to ZPL compatible ^GF-elements* with Golang and because of WASM even in the Browser: https://simonwaldherr.github.io/zplgfa/  

[![DOI](https://zenodo.org/badge/153820885.svg)](https://doi.org/10.5281/zenodo.15291211) 
[![GoDoc](https://godoc.org/github.com/SimonWaldherr/zplgfa?status.svg)](https://godoc.org/github.com/SimonWaldherr/zplgfa) 
[![Coverage Status](https://coveralls.io/repos/github/SimonWaldherr/zplgfa/badge.svg?branch=master)](https://coveralls.io/github/SimonWaldherr/zplgfa?branch=master) 
[![Go Report Card](https://goreportcard.com/badge/github.com/SimonWaldherr/zplgfa)](https://goreportcard.com/report/github.com/SimonWaldherr/zplgfa) 
[![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/SimonWaldherr/zplgfa/master/LICENSE) 

The ZPLGFA **Golang** package implements some functions to convert PNG, JPEG and GIF encoded graphic files to ZPL compatible ^GF-elements ([Graphic Fields](https://www.zebra.com/us/en/support-downloads/knowledge-articles/gf-graphic-field-zpl-command.html)).

If you need a ready to use application and don't want to hassle around with source code, take a look at the [ZPLGFA CLI Tool](https://github.com/SimonWaldherr/zplgfa/tree/master/cmd/zplgfa) which is based on this package.

You can also try the package directly in your browser &mdash; the [WebAssembly build](https://github.com/SimonWaldherr/zplgfa/tree/master/cmd/wasm) powers a [live demo with a converter and a minimal graphical editor](https://simonwaldherr.github.io/zplgfa/) (sources in [`docs/`](https://github.com/SimonWaldherr/zplgfa/tree/master/docs)).

## features

- convert `image.Image` values to complete ZPL labels with `ConvertToZPL`
- generate raw `^GF` graphic fields with `ConvertToGraphicField`
- choose between `ASCII`, `Binary`, `CompressedASCII` and `Z64` graphic field encodings
- decode ZPL `^GF` graphic fields back to black and white images with `ConvertZPLToImage`
- output black pixel runs as ZPL `^GB` line/box commands with `ConvertToZPLLines`
- flatten images with alpha transparency against a white background with `FlattenImage`
- compress ASCII graphic data with `CompressASCII`
- position graphics on the label with `ConvertToZPLAt`
- configure origin and reverse-field output with `ConvertToZPLWithOptions`
- decode and convert PNG, JPEG and GIF data directly from readers or files with `ConvertReaderToZPL` and `ConvertFileToZPL`

## install

1. [install Golang](https://golang.org/doc/install)
1. `go get simonwaldherr.de/go/zplgfa`

## example

take a look at the [example application](https://github.com/SimonWaldherr/zplgfa/tree/master/cmd/zplgfa)  
or at this sample code:  

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

    // output zpl with graphic field data to stdout
    fmt.Println(gfimg)
}

```

## package api

### Convert an image

```go
flat := zplgfa.FlattenImage(img)
zpl := zplgfa.ConvertToZPL(flat, zplgfa.CompressedASCII)
```

Use `zplgfa.Z64` when you want zlib-compressed, base64 encoded ZPL graphic data:

```go
zpl := zplgfa.ConvertToZPL(flat, zplgfa.Z64)
```

### Convert and position an image

```go
zpl := zplgfa.ConvertToZPLAt(flat, zplgfa.CompressedASCII, 120, 80)
```

### Convert with options

```go
zpl := zplgfa.ConvertToZPLWithOptions(flat, zplgfa.ConvertOptions{
    GraphicType: zplgfa.CompressedASCII,
    X:           120,
    Y:           80,
    Reverse:     true,
})
```

### Convert from a reader or file

`ConvertReaderToZPL` and `ConvertFileToZPL` decode PNG, JPEG and GIF input, flatten the image and return a complete ZPL label:

```go
zplFromReader, err := zplgfa.ConvertReaderToZPL(reader, zplgfa.CompressedASCII)
zplFromFile, err := zplgfa.ConvertFileToZPL("label.png", zplgfa.CompressedASCII)
```

### Generate only a graphic field

```go
gf := zplgfa.ConvertToGraphicField(flat, zplgfa.ASCII)
```

### Convert ZPL graphics back to an image

```go
img, err := zplgfa.ConvertZPLToImage(zpl)
```

### Output lines instead of a graphic field

```go
zpl := zplgfa.ConvertToZPLLines(flat)
```

## test and benchmark

Run the full test suite:

```sh
go test ./...
```

Run benchmarks:

```sh
go test -bench=. ./...
```

## label server

If you have dozens of label printers in use and need to fill and print label templates, this tool will help you:  

[![SimonWaldherr/ups - GitHub](https://gh-card.dev/repos/SimonWaldherr/ups.svg?fullname)](https://github.com/SimonWaldherr/ups)


## License

[MIT](https://github.com/SimonWaldherr/zplgfa/blob/master/LICENSE)

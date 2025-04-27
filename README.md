*since I am currently working exclusively in the home office and no longer have to do with labels professionally (only as a hobby), I can unfortunately no longer work on this project.*
*But if someone would like to provide me a [@Zebra](https://github.com/Zebra) printer, I would be happy to develop it further.*
*Of course, pull requests are still welcome.*

# ZPLGFA Golang Package

*convert pictures to ZPL compatible ^GF-elements*

[![DOI](https://zenodo.org/badge/153820885.svg)](https://doi.org/10.5281/zenodo.15291211) 
[![GoDoc](https://godoc.org/github.com/SimonWaldherr/zplgfa?status.svg)](https://godoc.org/github.com/SimonWaldherr/zplgfa) 
[![Coverage Status](https://coveralls.io/repos/github/SimonWaldherr/zplgfa/badge.svg?branch=master)](https://coveralls.io/github/SimonWaldherr/zplgfa?branch=master) 
[![Go Report Card](https://goreportcard.com/badge/github.com/SimonWaldherr/zplgfa)](https://goreportcard.com/report/github.com/SimonWaldherr/zplgfa) 
[![codebeat badge](https://codebeat.co/badges/28d795af-6f9b-453a-94c2-4fafb8b5b0d5)](https://codebeat.co/projects/github-com-simonwaldherr-zplgfa-master) 
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2FSimonWaldherr%2Fzplgfa.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2FSimonWaldherr%2Fzplgfa?ref=badge_shield) 
[![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/SimonWaldherr/zplgfa/master/LICENSE) 

The ZPLGFA **Golang** package implements some functions to convert PNG, JPEG and GIF encoded graphic files to ZPL compatible ^GF-elements ([Graphic Fields](https://www.zebra.com/us/en/support-downloads/knowledge-articles/gf-graphic-field-zpl-command.html)).

If you need a ready to use application and don't want to hassle around with source code, take a look at the [ZPLGFA CLI Tool](https://github.com/SimonWaldherr/zplgfa/tree/master/cmd/zplgfa) which is based on this package.

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

## label server

If you have dozens of label printers in use and need to fill and print label templates, this tool will help you:  

[![SimonWaldherr/ups - GitHub](https://gh-card.dev/repos/SimonWaldherr/ups.svg?fullname)](https://github.com/SimonWaldherr/ups)


## License

[MIT](https://github.com/SimonWaldherr/zplgfa/blob/master/LICENSE)

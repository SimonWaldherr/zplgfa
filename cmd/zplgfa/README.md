# ZPLGFA CLI Tool

The ZPLGFA cli tool converts PNG, JPEG and GIF images to [ZPL](https://www.zebra.com/content/dam/zebra/manuals/printers/common/programming/zpl-zbi2-pm-en.pdf) strings.
So if you need to print labels on a [ZPL](https://en.wikipedia.org/wiki/Zebra_(programming_language)) compatible printer
(like the amazing [ZEBRA ZM400](https://amzn.to/2OD5S4n)), but don't have ZPL-templates, you can use this free tool.

## install

1. [install Golang](https://golang.org/doc/install)
1. `go get simonwaldherr.de/go/zplgfa/cmd/zplgfa`

## usage

So if your image file is `label.png` and the IP of your printer is `192.168.178.42` you can print via this command:

```sh
zplgfa -file label.png | nc 192.168.178.42 9100
```

or via the integrated network capability:

```sh
zplgfa -file label.png -ip 192.168.178.42
```

You can also use some effects, e.g. blur:

```sh
zplgfa -file label.png -edit blur | nc 192.168.178.42 9100
```

or send special commands:

```sh
zplgfa -cmd feed -ip 192.168.178.42 -port 9100
```

you can also send multiple commands at once:


```sh
zplgfa -cmd cancel,calib,feed -ip 192.168.178.42 -port 9100
```

The ZPLGFA is actually just a demo application for the ZPLGFA package,
if you need something for productive work, look at the source and build something, depending on your needs

## test

You have your ZPLs, but no ZEBRA printer? You can test almost all ZPL functionality at [labelary.com](http://labelary.com/viewer.html).

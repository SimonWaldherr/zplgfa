//go:build js && wasm

// Package main builds a WebAssembly module that exposes the zplgfa
// conversion routines to JavaScript so the package can be used directly
// in the browser without a server roundtrip.
package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"syscall/js"

	"simonwaldherr.de/go/zplgfa"
)

// graphicTypeFromString maps a JS string to a zplgfa.GraphicType.
// Defaults to CompressedASCII for unknown values, matching the CLI tool.
func graphicTypeFromString(s string) zplgfa.GraphicType {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "ASCII":
		return zplgfa.ASCII
	case "BINARY":
		return zplgfa.Binary
	case "COMPRESSEDASCII", "":
		return zplgfa.CompressedASCII
	default:
		return zplgfa.CompressedASCII
	}
}

func isLineOutput(s string) bool {
	return strings.ToUpper(strings.TrimSpace(s)) == "LINES"
}

// jsUint8ArrayToBytes copies a JavaScript Uint8Array into a Go []byte.
func jsUint8ArrayToBytes(arr js.Value) []byte {
	length := arr.Get("length").Int()
	buf := make([]byte, length)
	js.CopyBytesToGo(buf, arr)
	return buf
}

// makeError produces a JS object {error: "..."} that callers can branch on.
func makeError(format string, a ...interface{}) map[string]interface{} {
	return map[string]interface{}{
		"error": fmt.Sprintf(format, a...),
	}
}

// convertImage is the main entry point exported to JavaScript.
//
// JS signature: zplgfaConvert(bytes: Uint8Array, graphicType?: string)
//
//	=> {zpl, width, height} | {error}
func convertImage(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return makeError("zplgfaConvert: expected at least one argument (Uint8Array)")
	}
	if args[0].IsNull() || args[0].IsUndefined() {
		return makeError("zplgfaConvert: image bytes argument is null/undefined")
	}

	data := jsUint8ArrayToBytes(args[0])
	if len(data) == 0 {
		return makeError("zplgfaConvert: image bytes are empty")
	}

	gt := zplgfa.CompressedASCII
	lines := false
	if len(args) >= 2 && args[1].Type() == js.TypeString {
		outputType := args[1].String()
		lines = isLineOutput(outputType)
		if !lines {
			gt = graphicTypeFromString(outputType)
		}
	}

	var zpl string
	if lines {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return makeError("zplgfaConvert: %s", err)
		}
		zpl = zplgfa.ConvertToZPLLines(zplgfa.FlattenImage(img))
	} else {
		var err error
		zpl, err = zplgfa.ConvertReaderToZPL(bytes.NewReader(data), gt)
		if err != nil {
			return makeError("zplgfaConvert: %s", err)
		}
	}

	// Decode the config again purely to expose width/height to the UI;
	// ignore errors here because the conversion above already succeeded.
	width, height := 0, 0
	if cfg, _, cerr := image.DecodeConfig(bytes.NewReader(data)); cerr == nil {
		width, height = cfg.Width, cfg.Height
	}

	return map[string]interface{}{
		"zpl":    zpl,
		"width":  width,
		"height": height,
	}
}

// convertRGBA accepts a flat RGBA byte buffer (4 bytes per pixel, row-major)
// plus width and height, allowing the in-browser editor to send canvas pixels
// directly without re-encoding to PNG first.
//
// JS signature: zplgfaConvertRGBA(rgba: Uint8Array, width: number, height: number, graphicType?: string)
//
//	=> {zpl, width, height} | {error}
func convertRGBA(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return makeError("zplgfaConvertRGBA: expected (rgba, width, height[, graphicType])")
	}
	width := args[1].Int()
	height := args[2].Int()
	if width <= 0 || height <= 0 {
		return makeError("zplgfaConvertRGBA: invalid dimensions %dx%d", width, height)
	}

	pixels := jsUint8ArrayToBytes(args[0])
	expected := width * height * 4
	if len(pixels) != expected {
		return makeError("zplgfaConvertRGBA: pixel buffer size %d does not match %dx%d*4=%d",
			len(pixels), width, height, expected)
	}

	img := &image.NRGBA{
		Pix:    pixels,
		Stride: 4 * width,
		Rect:   image.Rect(0, 0, width, height),
	}

	gt := zplgfa.CompressedASCII
	lines := false
	if len(args) >= 4 && args[3].Type() == js.TypeString {
		outputType := args[3].String()
		lines = isLineOutput(outputType)
		if !lines {
			gt = graphicTypeFromString(outputType)
		}
	}

	flat := zplgfa.FlattenImage(img)
	var zpl string
	if lines {
		zpl = zplgfa.ConvertToZPLLines(flat)
	} else {
		zpl = zplgfa.ConvertToZPL(flat, gt)
	}

	return map[string]interface{}{
		"zpl":    zpl,
		"width":  width,
		"height": height,
	}
}

func main() {
	js.Global().Set("zplgfaConvert", js.FuncOf(convertImage))
	js.Global().Set("zplgfaConvertRGBA", js.FuncOf(convertRGBA))

	// Signal readiness to the host page. The page can either listen for the
	// "zplgfaReady" event on document or poll for window.zplgfaReady === true.
	js.Global().Set("zplgfaReady", true)
	if doc := js.Global().Get("document"); !doc.IsUndefined() {
		event := js.Global().Get("Event").New("zplgfaReady")
		doc.Call("dispatchEvent", event)
	}

	// Block forever so the registered callbacks stay alive.
	select {}
}

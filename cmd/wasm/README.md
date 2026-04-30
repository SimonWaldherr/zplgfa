# zplgfa WebAssembly build

This directory contains a [WebAssembly](https://webassembly.org/) entry point
for the [`zplgfa`](https://github.com/SimonWaldherr/zplgfa) Go package. It lets
you convert images to ZPL `^GF` graphic fields directly in a browser, without
running a server.

A live demo is published from the [`docs/`](../../docs) folder via GitHub
Pages: <https://simonwaldherr.github.io/zplgfa/>.

## Building

You need Go 1.22 or newer.

```sh
GOOS=js GOARCH=wasm go build -o zplgfa.wasm ./cmd/wasm
```

The Go toolchain ships with the JavaScript glue file `wasm_exec.js`. Copy it
next to your `.html`:

```sh
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./wasm_exec.js   # Go ≥ 1.24
# or, on older Go:
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" ./wasm_exec.js
```

## Using from JavaScript

Once the module has been instantiated, two functions are registered on
`window`:

### `zplgfaConvert(bytes, graphicType?)`

Converts a PNG, JPEG or GIF buffer to ZPL.

* `bytes` — `Uint8Array` containing the encoded image.
* `graphicType` *(optional)* — one of `"CompressedASCII"` (default), `"ASCII"`
  or `"Binary"`.
* Returns `{ zpl, width, height }` on success, or `{ error }` on failure.

### `zplgfaConvertRGBA(rgba, width, height, graphicType?)`

Converts a raw RGBA pixel buffer (e.g. taken straight from a `<canvas>`) to
ZPL, skipping the encode/decode roundtrip.

* `rgba` — `Uint8Array` of length `width * height * 4`.
* `width`, `height` — dimensions in pixels.
* `graphicType` *(optional)* — same options as above.
* Returns `{ zpl, width, height }` or `{ error }`.

### Readiness

The page can wait for either of:

* `window.zplgfaReady === true`, or
* the `"zplgfaReady"` event dispatched on `document`.

### Minimal example

```html
<script src="wasm_exec.js"></script>
<script>
(async () => {
  const go = new Go();
  const { instance } = await WebAssembly.instantiateStreaming(
    fetch("zplgfa.wasm"), go.importObject);
  go.run(instance);

  await new Promise(r => {
    if (window.zplgfaReady) return r();
    document.addEventListener("zplgfaReady", () => r(), { once: true });
  });

  const resp = await fetch("logo.png");
  const bytes = new Uint8Array(await resp.arrayBuffer());
  const { zpl, error } = window.zplgfaConvert(bytes, "CompressedASCII");
  if (error) throw new Error(error);
  console.log(zpl);
})();
</script>
```

## Serving

Browsers will refuse to instantiate `.wasm` from `file://` URLs. Use any
static HTTP server, for example:

```sh
python3 -m http.server 8000
```

then open <http://localhost:8000/docs/>.

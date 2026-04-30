# zplgfa GitHub Pages demo

This folder is the source of the live ZPLGFA web demo published at
<https://simonwaldherr.github.io/zplgfa/>.

It is a fully static site that loads the
[`cmd/wasm`](../cmd/wasm) build of the `zplgfa` Go package and runs it
in the browser via WebAssembly. Nothing is uploaded to a server.

Files:

* `index.html` — UI markup with two tabs: "Converter" and "Mini editor".
* `app.js` — bootstraps the WASM module and powers the converter tab
  (file picker, drag & drop, resize / monochrome / invert, copy /
  download `.zpl`).
* `editor.js` — minimal canvas-based editor with pen, line, rectangle,
  ellipse, text, eraser, undo, image import and PNG / ZPL export.
* `style.css` — minimal dark theme.
* `wasm_exec.js` — Go runtime support, copied verbatim from
  `$(go env GOROOT)/lib/wasm/wasm_exec.js`.
* `zplgfa.wasm` — the prebuilt WebAssembly module. It is rebuilt by the
  `pages.yml` GitHub Actions workflow on every push to `master`.
* `.nojekyll` — disables Jekyll processing so files like `wasm_exec.js`
  are served as-is.

## Local preview

Browsers refuse to instantiate `.wasm` from `file://` URLs, so serve the
folder over HTTP, for example:

```sh
GOOS=js GOARCH=wasm go build -o docs/zplgfa.wasm ./cmd/wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" docs/wasm_exec.js
python3 -m http.server -d docs 8000
```

then open <http://localhost:8000/>.

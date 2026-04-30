// app.js — loads the zplgfa WebAssembly module and wires up the
// "Converter" tab (image file → ZPL).

(function () {
  "use strict";

  const statusEl = document.getElementById("status");
  const dropzone = document.getElementById("dropzone");
  const fileInput = document.getElementById("fileInput");
  const preview = document.getElementById("preview");
  const previewInfo = document.getElementById("previewInfo");
  const convertBtn = document.getElementById("convertBtn");
  const copyBtn = document.getElementById("copyBtn");
  const downloadBtn = document.getElementById("downloadBtn");
  const zplOutput = document.getElementById("zplOutput");
  const zplStats = document.getElementById("zplStats");

  const optType = document.getElementById("graphicType");
  const optResize = document.getElementById("resize");
  const optMono = document.getElementById("monochrome");
  const optInvert = document.getElementById("invert");

  // Tabs
  document.querySelectorAll("header nav .tab").forEach((tab) => {
    tab.addEventListener("click", () => {
      document.querySelectorAll("header nav .tab").forEach((t) => t.classList.remove("active"));
      document.querySelectorAll(".tab-panel").forEach((p) => p.classList.remove("active"));
      tab.classList.add("active");
      document.getElementById("tab-" + tab.dataset.tab).classList.add("active");
    });
  });

  // ---- WASM bootstrap ---------------------------------------------------
  let wasmReady = false;
  function setStatus(kind, msg) {
    statusEl.className = "status " + kind;
    statusEl.textContent = msg;
  }

  async function loadWasm() {
    if (typeof Go !== "function") {
      setStatus("error", "wasm_exec.js failed to load.");
      return;
    }
    const go = new Go();
    try {
      const resp = await fetch("zplgfa.wasm");
      if (!resp.ok) throw new Error("HTTP " + resp.status);
      let result;
      if (typeof WebAssembly.instantiateStreaming === "function") {
        result = await WebAssembly.instantiateStreaming(resp, go.importObject);
      } else {
        const buf = await resp.arrayBuffer();
        result = await WebAssembly.instantiate(buf, go.importObject);
      }
      go.run(result.instance); // resolves only on exit; we don't await it
      // Wait until the Go main() registered the globals.
      await new Promise((resolve) => {
        if (window.zplgfaReady) return resolve();
        document.addEventListener("zplgfaReady", () => resolve(), { once: true });
        // Fallback poll in case the event was missed during registration.
        const t = setInterval(() => {
          if (window.zplgfaReady) { clearInterval(t); resolve(); }
        }, 25);
      });
      wasmReady = true;
      setStatus("ready", "WebAssembly module loaded. Ready to convert.");
      convertBtn.disabled = !currentImageBytes;
      document.getElementById("exportZplBtn").disabled = false;
      document.dispatchEvent(new CustomEvent("zplgfa:loaded"));
    } catch (err) {
      setStatus("error", "Failed to load WASM: " + err.message);
      console.error(err);
    }
  }

  loadWasm();

  // ---- Converter tab ----------------------------------------------------
  let currentImageBytes = null; // Uint8Array of the selected file
  let currentImageEl = null;    // HTMLImageElement (for preview/manipulation)

  function readFile(file) {
    return new Promise((resolve, reject) => {
      const r = new FileReader();
      r.onerror = () => reject(r.error);
      r.onload = () => resolve(new Uint8Array(r.result));
      r.readAsArrayBuffer(file);
    });
  }

  function loadImageFromBytes(bytes, mime) {
    return new Promise((resolve, reject) => {
      const blob = new Blob([bytes], { type: mime || "application/octet-stream" });
      const url = URL.createObjectURL(blob);
      const img = new Image();
      img.onload = () => { URL.revokeObjectURL(url); resolve(img); };
      img.onerror = (e) => { URL.revokeObjectURL(url); reject(new Error("Could not decode image")); };
      img.src = url;
    });
  }

  async function handleFile(file) {
    if (!file) return;
    if (!/^image\/(png|jpeg|gif)$/.test(file.type)) {
      setStatus("error", "Unsupported file type: " + (file.type || "unknown"));
      return;
    }
    try {
      currentImageBytes = await readFile(file);
      currentImageEl = await loadImageFromBytes(currentImageBytes, file.type);
      drawPreview();
      previewInfo.textContent =
        file.name + " — " + currentImageEl.naturalWidth + "×" + currentImageEl.naturalHeight +
        " (" + Math.round(currentImageBytes.length / 1024) + " KiB)";
      convertBtn.disabled = !wasmReady;
      if (wasmReady) setStatus("ready", "Image loaded. Click “Convert to ZPL”.");
    } catch (err) {
      setStatus("error", "Could not load image: " + err.message);
    }
  }

  function drawPreview() {
    if (!currentImageEl) return;
    const factor = clampNumber(parseFloat(optResize.value), 0.05, 4) || 1;
    const w = Math.max(1, Math.round(currentImageEl.naturalWidth * factor));
    const h = Math.max(1, Math.round(currentImageEl.naturalHeight * factor));
    preview.width = w;
    preview.height = h;
    const ctx = preview.getContext("2d");
    ctx.imageSmoothingEnabled = true;
    ctx.clearRect(0, 0, w, h);
    ctx.drawImage(currentImageEl, 0, 0, w, h);

    if (optMono.checked || optInvert.checked) {
      const data = ctx.getImageData(0, 0, w, h);
      applyPixelOps(data, optMono.checked, optInvert.checked);
      ctx.putImageData(data, 0, 0);
    }
  }

  function applyPixelOps(imgData, mono, invert) {
    const px = imgData.data;
    for (let i = 0; i < px.length; i += 4) {
      let r = px[i], g = px[i + 1], b = px[i + 2];
      const a = px[i + 3];
      // Composite onto white using alpha.
      if (a < 255) {
        const k = a / 255;
        r = Math.round(r * k + 255 * (1 - k));
        g = Math.round(g * k + 255 * (1 - k));
        b = Math.round(b * k + 255 * (1 - k));
        px[i + 3] = 255;
      }
      if (invert) { r = 255 - r; g = 255 - g; b = 255 - b; }
      if (mono) {
        // Luma-based threshold at 128 (ITU-R BT.601 coefficients).
        const y = 0.299 * r + 0.587 * g + 0.114 * b;
        const v = y < 128 ? 0 : 255;
        r = g = b = v;
      }
      px[i] = r; px[i + 1] = g; px[i + 2] = b;
    }
  }

  function clampNumber(n, lo, hi) {
    if (Number.isNaN(n)) return NaN;
    return Math.min(hi, Math.max(lo, n));
  }

  // Convert canvas → PNG blob → bytes → wasm so any UI options applied to
  // the preview canvas (resize, monochrome, invert) flow into ZPL output.
  async function canvasToPngBytes(canvas) {
    return new Promise((resolve, reject) => {
      canvas.toBlob((blob) => {
        if (!blob) return reject(new Error("toBlob() returned null"));
        const r = new FileReader();
        r.onerror = () => reject(r.error);
        r.onload = () => resolve(new Uint8Array(r.result));
        r.readAsArrayBuffer(blob);
      }, "image/png");
    });
  }

  async function convert() {
    if (!wasmReady) return;
    if (!currentImageEl) {
      setStatus("error", "Pick an image first.");
      return;
    }
    try {
      drawPreview();
      const bytes = await canvasToPngBytes(preview);
      const result = window.zplgfaConvert(bytes, optType.value);
      if (result && result.error) throw new Error(result.error);
      const zpl = result.zpl;
      zplOutput.value = zpl;
      zplStats.textContent = zpl.length.toLocaleString() + " characters · " +
        (result.width || preview.width) + "×" + (result.height || preview.height);
      copyBtn.disabled = false;
      downloadBtn.disabled = false;
      setStatus("ready", "Conversion successful.");
      // Enable and optionally trigger preview.
      const convPreviewBtn = document.getElementById("convPreviewBtn");
      if (convPreviewBtn) convPreviewBtn.disabled = false;
      if (document.getElementById("convAutoPreview") && document.getElementById("convAutoPreview").checked) {
        const el = document.getElementById("convPreviewResult");
        if (el) renderZplPreview(zpl,
          document.getElementById("convPreviewDpmm").value,
          document.getElementById("convPreviewLabelW").value,
          document.getElementById("convPreviewLabelH").value,
          el);
      }
    } catch (err) {
      setStatus("error", "Conversion failed: " + err.message);
    }
  }

  // ---- Wiring -----------------------------------------------------------
  fileInput.addEventListener("change", (e) => handleFile(e.target.files[0]));

  ["dragenter", "dragover"].forEach((evt) =>
    dropzone.addEventListener(evt, (e) => {
      e.preventDefault(); e.stopPropagation();
      dropzone.classList.add("drag");
    })
  );
  ["dragleave", "drop"].forEach((evt) =>
    dropzone.addEventListener(evt, (e) => {
      e.preventDefault(); e.stopPropagation();
      dropzone.classList.remove("drag");
    })
  );
  dropzone.addEventListener("drop", (e) => {
    if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFile(e.dataTransfer.files[0]);
    }
  });

  [optResize, optMono, optInvert].forEach((el) =>
    el.addEventListener("input", drawPreview));

  convertBtn.addEventListener("click", convert);

  copyBtn.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(zplOutput.value);
      copyBtn.textContent = "Copied!";
      setTimeout(() => (copyBtn.textContent = "Copy"), 1200);
    } catch (err) {
      // Fallback for older browsers.
      zplOutput.select();
      document.execCommand("copy");
    }
  });

  downloadBtn.addEventListener("click", () => {
    const blob = new Blob([zplOutput.value], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "label.zpl";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  });

  // ---- ZPL Preview (shared helper) -------------------------------------
  // Sends ZPL to the Labelary rendering API and displays the resulting image.
  async function renderZplPreview(zplText, dpmm, widthIn, heightIn, resultEl) {
    if (!zplText || !zplText.trim()) {
      resultEl.innerHTML = '<p class="hint">No ZPL to preview.</p>';
      return;
    }
    resultEl.innerHTML = '<p class="hint preview-spinner">Rendering&hellip;</p>';
    const url =
      "https://api.labelary.com/v1/printers/" +
      encodeURIComponent(dpmm) + "dpmm/labels/" +
      encodeURIComponent(widthIn) + "x" +
      encodeURIComponent(heightIn) + "/0/";
    try {
      const resp = await fetch(url, {
        method: "POST",
        headers: { "Accept": "image/png" },
        body: zplText,
      });
      if (!resp.ok) throw new Error("Labelary API request failed with HTTP " + resp.status);
      const blob = await resp.blob();
      const imgUrl = URL.createObjectURL(blob);
      resultEl.innerHTML = "";
      const img = document.createElement("img");
      img.src = imgUrl;
      img.alt = "ZPL label preview";
      img.className = "zpl-preview-img";
      img.onload = () => URL.revokeObjectURL(imgUrl);
      resultEl.appendChild(img);
    } catch (err) {
      resultEl.innerHTML =
        '<p class="hint preview-error">Preview failed: ' +
        err.message +
        ". Check your network connection.</p>";
    }
  }

  // Expose a tiny API for editor.js to share the wasm-ready signal.
  window.__zplgfaApp = {
    isReady: () => wasmReady,
    setStatus,
    renderZplPreview,
  };

  // ---- ZPL Preview (Converter tab) -------------------------------------
  const convPreviewDpmm = document.getElementById("convPreviewDpmm");
  const convPreviewLabelW = document.getElementById("convPreviewLabelW");
  const convPreviewLabelH = document.getElementById("convPreviewLabelH");
  const convAutoPreview = document.getElementById("convAutoPreview");
  const convPreviewBtn = document.getElementById("convPreviewBtn");
  const convPreviewResult = document.getElementById("convPreviewResult");

  convPreviewBtn.addEventListener("click", () =>
    renderZplPreview(
      zplOutput.value,
      convPreviewDpmm.value,
      convPreviewLabelW.value,
      convPreviewLabelH.value,
      convPreviewResult
    )
  );
})();

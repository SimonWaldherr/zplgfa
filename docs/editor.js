// editor.js — minimal in-browser graphical editor that exports its
// canvas content directly to ZPL via the WASM module.

(function () {
  "use strict";

  const canvas = document.getElementById("editor");
  const ctx = canvas.getContext("2d", { willReadFrequently: true });

  const canvasW = document.getElementById("canvasW");
  const canvasH = document.getElementById("canvasH");
  const resizeCanvasBtn = document.getElementById("resizeCanvasBtn");
  const toolSel = document.getElementById("tool");
  const brushSize = document.getElementById("brushSize");
  const brushSizeValue = document.getElementById("brushSizeValue");
  const textInput = document.getElementById("textInput");
  const fontSize = document.getElementById("fontSize");
  const clearBtn = document.getElementById("clearBtn");
  const undoBtn = document.getElementById("undoBtn");
  const importInput = document.getElementById("importInput");

  const exportZplBtn = document.getElementById("exportZplBtn");
  const exportPngBtn = document.getElementById("exportPngBtn");
  const editorGraphicType = document.getElementById("editorGraphicType");
  const editorThreshold = document.getElementById("editorThreshold");
  const autoExportCheckbox = document.getElementById("autoExport");

  const zplOut = document.getElementById("editorZplOutput");
  const zplStats = document.getElementById("editorZplStats");
  const copyBtn = document.getElementById("editorCopyBtn");
  const downloadBtn = document.getElementById("editorDownloadBtn");

  const previewDpmm = document.getElementById("previewDpmm");
  const previewLabelW = document.getElementById("previewLabelW");
  const previewLabelH = document.getElementById("previewLabelH");
  const autoPreviewCheckbox = document.getElementById("autoPreview");
  const editorPreviewBtn = document.getElementById("editorPreviewBtn");
  const editorPreviewResult = document.getElementById("editorPreviewResult");

  // Stack of ImageData snapshots for undo.
  const undoStack = [];
  const UNDO_LIMIT = 30;

  function pushUndo() {
    try {
      const snap = ctx.getImageData(0, 0, canvas.width, canvas.height);
      undoStack.push(snap);
      if (undoStack.length > UNDO_LIMIT) undoStack.shift();
    } catch (_) { /* ignore (e.g. tainted canvas) */ }
  }

  function clearCanvas() {
    pushUndo();
    ctx.fillStyle = "#ffffff";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
  }

  function resizeCanvas(w, h) {
    w = Math.max(16, Math.min(2000, w | 0));
    h = Math.max(16, Math.min(2000, h | 0));
    // Preserve existing pixels by drawing onto a new buffer.
    const old = document.createElement("canvas");
    old.width = canvas.width;
    old.height = canvas.height;
    old.getContext("2d").drawImage(canvas, 0, 0);
    canvas.width = w;
    canvas.height = h;
    ctx.fillStyle = "#ffffff";
    ctx.fillRect(0, 0, w, h);
    ctx.drawImage(old, 0, 0);
    undoStack.length = 0;
    syncDimsFromCanvas();
  }

  // Initialize white background.
  clearCanvas();
  undoStack.length = 0; // initial blank doesn't count

  // ---- Drawing state ---------------------------------------------------
  let drawing = false;
  let startX = 0, startY = 0;
  let lastX = 0, lastY = 0;
  let preDragSnapshot = null;

  function pointerPos(evt) {
    const rect = canvas.getBoundingClientRect();
    const sx = canvas.width / rect.width;
    const sy = canvas.height / rect.height;
    return {
      x: Math.round((evt.clientX - rect.left) * sx),
      y: Math.round((evt.clientY - rect.top) * sy),
    };
  }

  function strokeStyle() {
    const tool = toolSel.value;
    const w = parseInt(brushSize.value, 10) || 1;
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    ctx.lineWidth = w;
    ctx.strokeStyle = tool === "erase" ? "#ffffff" : "#000000";
    ctx.fillStyle = tool === "erase" ? "#ffffff" : "#000000";
  }

  function onPointerDown(e) {
    e.preventDefault();
    canvas.setPointerCapture && canvas.setPointerCapture(e.pointerId);
    pushUndo();
    const { x, y } = pointerPos(e);
    startX = lastX = x;
    startY = lastY = y;
    drawing = true;
    strokeStyle();

    const tool = toolSel.value;
    if (tool === "pen" || tool === "erase") {
      ctx.beginPath();
      ctx.moveTo(x, y);
      ctx.lineTo(x + 0.01, y + 0.01); // ensures a dot for a single click
      ctx.stroke();
    } else if (tool === "text") {
      const size = Math.max(6, parseInt(fontSize.value, 10) || 24);
      ctx.font = size + "px sans-serif";
      ctx.textBaseline = "top";
      ctx.fillStyle = "#000000";
      ctx.fillText(textInput.value || "", x, y);
      drawing = false;
      scheduleAutoExport();
    } else {
      // Shape tools draw on move using a saved snapshot.
      preDragSnapshot = ctx.getImageData(0, 0, canvas.width, canvas.height);
    }
  }

  function onPointerMove(e) {
    if (!drawing) return;
    const { x, y } = pointerPos(e);
    const tool = toolSel.value;
    strokeStyle();

    if (tool === "pen" || tool === "erase") {
      ctx.beginPath();
      ctx.moveTo(lastX, lastY);
      ctx.lineTo(x, y);
      ctx.stroke();
      lastX = x; lastY = y;
      return;
    }

    if (preDragSnapshot) ctx.putImageData(preDragSnapshot, 0, 0);

    if (tool === "line") {
      ctx.beginPath();
      ctx.moveTo(startX, startY);
      ctx.lineTo(x, y);
      ctx.stroke();
    } else if (tool === "rect") {
      ctx.strokeRect(
        Math.min(startX, x), Math.min(startY, y),
        Math.abs(x - startX), Math.abs(y - startY)
      );
    } else if (tool === "rectFill") {
      ctx.fillRect(
        Math.min(startX, x), Math.min(startY, y),
        Math.abs(x - startX), Math.abs(y - startY)
      );
    } else if (tool === "ellipse") {
      const cx = (startX + x) / 2;
      const cy = (startY + y) / 2;
      const rx = Math.abs(x - startX) / 2;
      const ry = Math.abs(y - startY) / 2;
      ctx.beginPath();
      ctx.ellipse(cx, cy, rx, ry, 0, 0, Math.PI * 2);
      ctx.stroke();
    }
  }

  function onPointerUp() {
    if (drawing) {
      drawing = false;
      preDragSnapshot = null;
      scheduleAutoExport();
    }
  }

  canvas.addEventListener("pointerdown", onPointerDown);
  canvas.addEventListener("pointermove", onPointerMove);
  window.addEventListener("pointerup", onPointerUp);

  // ---- UI wiring -------------------------------------------------------
  brushSize.addEventListener("input", () => {
    brushSizeValue.textContent = brushSize.value;
  });
  resizeCanvasBtn.addEventListener("click", () =>
    resizeCanvas(parseInt(canvasW.value, 10), parseInt(canvasH.value, 10))
  );
  clearBtn.addEventListener("click", () => {
    clearCanvas();
    scheduleAutoExport();
  });
  undoBtn.addEventListener("click", () => {
    const snap = undoStack.pop();
    if (snap) ctx.putImageData(snap, 0, 0);
    scheduleAutoExport();
  });

  importInput.addEventListener("change", (e) => {
    const file = e.target.files && e.target.files[0];
    if (!file) return;
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      pushUndo();
      // Fit imported image inside the current canvas, centered, preserving aspect.
      const ratio = Math.min(canvas.width / img.naturalWidth, canvas.height / img.naturalHeight, 1);
      const w = Math.max(1, Math.round(img.naturalWidth * ratio));
      const h = Math.max(1, Math.round(img.naturalHeight * ratio));
      const dx = Math.round((canvas.width - w) / 2);
      const dy = Math.round((canvas.height - h) / 2);
      ctx.drawImage(img, dx, dy, w, h);
      URL.revokeObjectURL(url);
      scheduleAutoExport();
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      alert("Could not load image.");
    };
    img.src = url;
    // Allow re-importing the same file later.
    importInput.value = "";
  });

  // ---- Export ----------------------------------------------------------
  function applyThresholdInPlace(imgData) {
    const px = imgData.data;
    for (let i = 0; i < px.length; i += 4) {
      // Already opaque since the canvas background is solid white.
      // Luma using ITU-R BT.601 coefficients, then threshold at 128.
      const y = 0.299 * px[i] + 0.587 * px[i + 1] + 0.114 * px[i + 2];
      const v = y < 128 ? 0 : 255;
      px[i] = px[i + 1] = px[i + 2] = v;
      px[i + 3] = 255;
    }
  }

  function doExportZpl() {
    if (!window.zplgfaConvertRGBA) return false;
    const w = canvas.width;
    const h = canvas.height;
    const data = ctx.getImageData(0, 0, w, h);
    if (editorThreshold.checked) applyThresholdInPlace(data);
    const buf = new Uint8Array(data.data.buffer.slice(0));
    const result = window.zplgfaConvertRGBA(buf, w, h, editorGraphicType.value);
    if (result && result.error) {
      alert("Export failed: " + result.error);
      return false;
    }
    zplOut.value = result.zpl;
    zplStats.textContent = result.zpl.length.toLocaleString() + " characters · " + w + "×" + h;
    copyBtn.disabled = false;
    downloadBtn.disabled = false;
    editorPreviewBtn.disabled = false;
    return true;
  }

  // Debounced auto-export triggered after drawing operations.
  let autoExportTimer = null;
  function scheduleAutoExport() {
    if (!autoExportCheckbox.checked) return;
    if (!window.zplgfaConvertRGBA) return;
    clearTimeout(autoExportTimer);
    autoExportTimer = setTimeout(() => {
      if (doExportZpl() && autoPreviewCheckbox.checked) {
        triggerPreview();
      }
    }, 600);
  }

  exportZplBtn.addEventListener("click", () => {
    if (!window.zplgfaConvertRGBA) {
      alert("WASM module not ready yet.");
      return;
    }
    if (doExportZpl() && autoPreviewCheckbox.checked) {
      triggerPreview();
    }
  });

  // Enable button and run initial export once WASM is loaded.
  document.addEventListener("zplgfa:loaded", () => {
    exportZplBtn.disabled = false;
    if (autoExportCheckbox.checked) doExportZpl();
  });

  exportPngBtn.addEventListener("click", () => {
    canvas.toBlob((blob) => {
      if (!blob) return;
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "editor.png";
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }, "image/png");
  });

  copyBtn.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(zplOut.value);
      copyBtn.textContent = "Copied!";
      setTimeout(() => (copyBtn.textContent = "Copy"), 1200);
    } catch (_) {
      zplOut.select();
      document.execCommand("copy");
    }
  });

  downloadBtn.addEventListener("click", () => {
    const blob = new Blob([zplOut.value], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "editor.zpl";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  });

  // ---- ZPL Preview (Editor tab) ----------------------------------------
  // Auto-compute label dimensions (inches) from canvas size and selected DPI.
  const MM_PER_INCH = 25.4;
  const QUARTER_INCH_PRECISION = 4; // round to nearest 0.25 in
  function syncDimsFromCanvas() {
    const dpmm = parseInt(previewDpmm.value, 10) || 8;
    const dotsPerInch = dpmm * MM_PER_INCH;
    previewLabelW.value = Math.max(0.5, Math.round((canvas.width / dotsPerInch) * QUARTER_INCH_PRECISION) / QUARTER_INCH_PRECISION).toFixed(2);
    previewLabelH.value = Math.max(0.5, Math.round((canvas.height / dotsPerInch) * QUARTER_INCH_PRECISION) / QUARTER_INCH_PRECISION).toFixed(2);
  }

  // Recompute on DPI change.
  previewDpmm.addEventListener("change", syncDimsFromCanvas);
  // Initial sync.
  syncDimsFromCanvas();

  function triggerPreview() {
    const app = window.__zplgfaApp;
    if (!app || !app.renderZplPreview) return;
    app.renderZplPreview(
      zplOut.value,
      previewDpmm.value,
      previewLabelW.value,
      previewLabelH.value,
      editorPreviewResult
    );
  }

  editorPreviewBtn.addEventListener("click", triggerPreview);
})();

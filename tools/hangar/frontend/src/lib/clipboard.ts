import { Clipboard } from "@wailsio/runtime";

// copyText copies text to the clipboard. The packaged Wails webview doesn't
// expose a working navigator.clipboard (no secure-context permission), so the
// Wails runtime clipboard is tried first; the browser paths are fallbacks for
// running the frontend with `vite` in a real browser during dev.
export async function copyText(text: string): Promise<void> {
  // 1) Wails runtime — the reliable path inside the app.
  try {
    await Clipboard.SetText(text);
    return;
  } catch {
    // fall through
  }
  // 2) Async clipboard API (browser dev).
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return;
    }
  } catch {
    // fall through
  }
  // 3) Legacy execCommand fallback.
  const ta = document.createElement("textarea");
  ta.value = text;
  ta.style.position = "fixed";
  ta.style.opacity = "0";
  ta.setAttribute("readonly", "");
  document.body.appendChild(ta);
  ta.select();
  try {
    if (!document.execCommand("copy")) {
      throw new Error("copy command was rejected");
    }
  } finally {
    document.body.removeChild(ta);
  }
}

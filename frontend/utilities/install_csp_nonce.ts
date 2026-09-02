import getCSPNonce from "./nonce";

// Third-party libraries (react-tooltip v4/v5, sonner, Emotion, Ace) inject
// <style> elements at runtime via document.createElement. Under a strict CSP
// (style-src 'self' 'nonce-…') those tags are blocked unless they carry the
// nonce, and several of these libraries expose no nonce option. Wrapping
// document.createElement to stamp the server's nonce on created <style> tags
// covers all of them at the single point they share. No-op when the page is
// served without a CSP.
//
// Scoped to <style> deliberately: every runtime CSP violation we hit is an
// inline style, same-origin <script src> chunks are already allowed by
// script-src 'self', and the template's inline scripts are nonced server-side.
// So Fleet never needs to auto-nonce a <script>, and not doing so keeps this
// wrapper from ever stamping the higher-value target. This only runs for code
// that is already executing JS, so it is not itself an injection vector, and
// the nonce is readable from the csp-nonce meta tag regardless.

// "disabled" is the sentinel nonce the server emits when no CSP is active.
const CSP_DISABLED = "disabled";

export const installCSPNonce = (): void => {
  const nonce = getCSPNonce();
  if (!nonce || nonce === CSP_DISABLED) {
    return;
  }

  const original = document.createElement.bind(document);
  document.createElement = function createElement(
    tagName: string,
    options?: ElementCreationOptions
  ) {
    const element = original(tagName, options);
    if (typeof tagName === "string" && tagName.toLowerCase() === "style") {
      element.setAttribute("nonce", nonce);
    }
    return element;
  } as typeof document.createElement;
};

export default installCSPNonce;

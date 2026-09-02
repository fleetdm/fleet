import getCSPNonce from "./nonce";

// Third-party libraries (react-tooltip v4/v5, sonner, Emotion, Ace) inject
// <style> elements at runtime via document.createElement. Under a strict CSP
// (style-src 'self' 'nonce-…') those tags are blocked unless they carry the
// nonce, and several of these libraries expose no nonce option. Wrapping
// document.createElement so every dynamically created <style>/<script> is
// stamped with the server's nonce before insertion covers all of them at the
// single point they share. No-op when the page is served without a CSP.
const NONCE_TAGS = new Set(["style", "script"]);

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
    if (typeof tagName === "string" && NONCE_TAGS.has(tagName.toLowerCase())) {
      element.setAttribute("nonce", nonce);
    }
    return element;
  } as typeof document.createElement;
};

export default installCSPNonce;

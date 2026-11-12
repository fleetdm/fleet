import getCSPNonce from "./nonce";

// Third-party libraries (react-tooltip v4/v5, sonner, Emotion, Ace) inject
// <style> elements at runtime. Under a strict CSP (style-src 'self' 'nonce-…')
// those tags are blocked unless they carry the nonce, and several of these
// libraries expose no nonce option. Wrap the two DOM APIs they create styles
// through — document.createElement (React-based libs) and
// document.createElementNS (Ace) — to stamp the server's nonce on every created
// <style>. No-op when the page is served without a CSP.
//
// Scoped to <style> deliberately: every runtime CSP violation we hit is an
// inline style, same-origin <script src> chunks are already allowed by
// script-src 'self', and the template's inline scripts are nonced server-side.
// So Fleet never needs to auto-nonce a <script>, and not doing so keeps this
// wrapper from ever stamping the higher-value target. This only runs for code
// that is already executing JS, so it is not itself an injection vector, and
// the nonce is readable from the csp-nonce meta tag regardless.
//
// This module self-installs on import and must be the FIRST import of the entry
// so it runs before any library. Some libraries (e.g. sonner, Ace) inject their
// base <style> at module-load time, so installing later — even before the first
// render — is too late for those.

// "disabled" is the sentinel nonce the server emits when no CSP is active.
const CSP_DISABLED = "disabled";

export const installCSPNonce = (): void => {
  const nonce = getCSPNonce();
  if (!nonce || nonce === CSP_DISABLED) {
    return;
  }

  const stampStyle = <T extends Element>(element: T, tagName: unknown): T => {
    if (typeof tagName === "string" && tagName.toLowerCase() === "style") {
      element.setAttribute("nonce", nonce);
    }
    return element;
  };

  const originalCreate = document.createElement.bind(document);
  document.createElement = function createElement(
    tagName: string,
    options?: ElementCreationOptions
  ) {
    return stampStyle(originalCreate(tagName, options), tagName);
  } as typeof document.createElement;

  const originalCreateNS = document.createElementNS.bind(document);
  document.createElementNS = function createElementNS(
    namespaceURI: string | null,
    qualifiedName: string,
    options?: ElementCreationOptions
  ) {
    return stampStyle(
      originalCreateNS(namespaceURI, qualifiedName, options),
      qualifiedName
    );
  } as typeof document.createElementNS;
};

installCSPNonce();

export default installCSPNonce;

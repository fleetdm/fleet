// Reads the per-response CSP nonce the server injects into the served HTML
// (see `react.tmpl`'s `<meta property="csp-nonce">`). Runtime style/script
// injectors (Emotion, webpack chunk loading) must stamp this nonce so a strict
// `style-src`/`script-src` policy doesn't block them. Returns "" when the page
// was served without a nonce.
const getCSPNonce = (): string => {
  if (typeof document === "undefined") {
    return "";
  }
  return (
    document
      .querySelector('meta[property="csp-nonce"]')
      ?.getAttribute("content") ?? ""
  );
};

export default getCSPNonce;

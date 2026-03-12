import { createRoot } from "react-dom/client";
import createCache from "@emotion/cache";

// used for babel polyfills.
import "core-js/stable";
import "regenerator-runtime/runtime";

import "./public-path";
import routes from "./router";
import "./index.scss";

// Read CSP nonce from meta tag set by server (for webpack dynamic chunk loading)
const cspNonceMeta = document.querySelector('meta[property="csp-nonce"]');
if (cspNonceMeta) {
  // eslint-disable-next-line no-undef
  __webpack_nonce__ = cspNonceMeta.getAttribute("content");
}

// eslint-disable-next-line import/prefer-default-export
export const emotionCache = createCache({
  key: "emotion-css",
  nonce: cspNonceMeta.getAttribute("content"),
});

if (typeof window !== "undefined") {
  const { document } = global;
  const app = document.getElementById("app");
  const root = createRoot(app);
  root.render(routes);
}

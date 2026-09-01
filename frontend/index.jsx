import React from "react";
import { createRoot } from "react-dom/client";
import { CacheProvider } from "@emotion/react";
import createCache from "@emotion/cache";

// used for babel polyfills.
import "core-js/stable";
import "regenerator-runtime/runtime";

// Base styles for the legacy react-select v1 Dropdown. Must load before
// index.scss so Fleet's overrides win, and from the entry so every route has
// them — route chunks would otherwise only pull them in on some pages.
import "react-select/dist/react-select.css";

import "./public-path";
import routes from "./router";
import "./index.scss";
import { initTheme } from "./utilities/theme";
import getCSPNonce from "./utilities/nonce";

// react-select v5 and other Emotion consumers inject <style> tags at runtime.
// Route them through a cache carrying the server's CSP nonce so a strict
// style-src doesn't block them.
const emotionCache = createCache({ key: "css", nonce: getCSPNonce() });

if (typeof window !== "undefined") {
  initTheme();
  const { document } = global;
  const app = document.getElementById("app");
  const root = createRoot(app);
  root.render(<CacheProvider value={emotionCache}>{routes}</CacheProvider>);
}

import { createRoot } from "react-dom/client";

// used for babel polyfills.
import "core-js/stable";
import "regenerator-runtime/runtime";

import "./public-path";
import routes from "./router";
import "./index.scss";

if (typeof window !== "undefined") {
  const { document } = global;

  // Set mobile-friendly viewport for device-facing pages
  if (window.location.pathname.includes("/device/")) {
    document
      .getElementById("viewport-meta-tag")
      ?.setAttribute("content", "width=device-width, initial-scale=1.0");
  }

  const app = document.getElementById("app");
  const root = createRoot(app);
  root.render(routes);
}

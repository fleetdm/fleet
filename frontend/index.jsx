import { createRoot } from "react-dom/client";

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

if (typeof window !== "undefined") {
  initTheme();
  const { document } = global;
  const app = document.getElementById("app");
  const root = createRoot(app);
  root.render(routes);
}

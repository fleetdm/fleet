import { createRoot } from "react-dom/client";

// used for babel polyfills.
import "core-js/stable";
import "regenerator-runtime/runtime";

import "./public-path";
import routes from "./router";
// Base layout/positioning CSS for react-select v1 (used by the Dropdown
// component). Imported here — in the main entry, never code-split — so it's
// always loaded, and evaluated before index.scss so Fleet's dark-mode
// .Select-value-label override (same specificity) wins the cascade tie.
import "react-select/dist/react-select.css";
import "./index.scss";
import { initTheme } from "./utilities/theme";

if (typeof window !== "undefined") {
  initTheme();
  const { document } = global;
  const app = document.getElementById("app");
  const root = createRoot(app);
  root.render(routes);
}

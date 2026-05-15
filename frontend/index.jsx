import { createRoot } from "react-dom/client";

// used for babel polyfills.
import "core-js/stable";
import "regenerator-runtime/runtime";

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

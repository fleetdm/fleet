import ReactDOM from "react-dom";

// used for babel polyfills.
import "core-js/stable";
import "regenerator-runtime/runtime";

import "./public-path";
import routes from "./router";
import "./index.scss";

if (typeof window !== "undefined") {
  const { document } = global;
  const app = document.getElementById("app");

  ReactDOM.render(routes, app);
}

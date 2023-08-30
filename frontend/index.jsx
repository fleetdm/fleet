import ReactDOM from "react-dom";

// used for babel polyfills.
import "core-js/stable";
import "regenerator-runtime/runtime";

import "./public-path";
import routes from "./router";
import "./index.scss";
import { init as initApm } from '@elastic/apm-rum'

const apm = initApm({
  // Set required service name (allowed characters: a-z, A-Z, 0-9, -, _, and space)
  serviceName: APMService,

  // Set custom APM Server URL (default: http://localhost:8200)
  serverUrl: APMServer,
})

if (typeof window !== "undefined") {
  const { document } = global;
  const app = document.getElementById("app");

  ReactDOM.render(routes, app);
}

import { merge } from "webpack-merge";
import * as common from "./webpack.common.js";

export default merge(common, {
  mode: "development",
  devtool: "inline-source-map",
});

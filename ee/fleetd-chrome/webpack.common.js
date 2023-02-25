import { resolve as _resolve, dirname } from "path";
import { fileURLToPath } from "url";
import CopyWebpackPlugin from "copy-webpack-plugin";
import HtmlWebpackPlugin from "html-webpack-plugin";

const __dirname = dirname(fileURLToPath(import.meta.url));

export const entry = {
  background: "./src/background.ts",
  popup: "./src/popup.ts",
};
export const module = {
  rules: [
    {
      test: /\.tsx?$/,
      use: "ts-loader",
      exclude: /node_modules/,
    },
  ],
};
export const resolve = {
  extensions: [".tsx", ".ts", ".js"],
};
export const plugins = [
  new CopyWebpackPlugin({
    patterns: [
      { from: "./src/manifest.json" },
      { from: "./src/schema.json" },
      { from: "./src/icons" },
    ],
  }),
  new HtmlWebpackPlugin({
    template: "./src/popup.html",
    filename: "popup.html",
    chunks: ["popup"],
  }),
];
export const output = {
  filename: "[name].bundle.js",
  path: _resolve(__dirname, "dist"),
  chunkFormat: "commonjs",
  clean: true,
};

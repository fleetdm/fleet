import { resolve as _resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

export const entry = {
  background: "./src/background.ts",
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
export const output = {
  filename: "[name].bundle.js",
  path: _resolve(__dirname, "dist", "js"),
  chunkFormat: "commonjs",
};

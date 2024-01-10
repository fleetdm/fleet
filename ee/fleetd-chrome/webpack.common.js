// Some of this webpack config inspired by MIT licensed
// https://github.com/samuelsimoes/chrome-extension-webpack-boilerplate

import { resolve as _resolve, dirname } from "path";
import { fileURLToPath } from "url";
import CopyWebpackPlugin from "copy-webpack-plugin";
import HtmlWebpackPlugin from "html-webpack-plugin";
import TerserPlugin from "terser-webpack-plugin";
import webpack from "webpack";
import dotenv from "dotenv";

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
  new webpack.DefinePlugin({
    ...Object.entries(dotenv.config().parsed).reduce(
      (acc, curr) => ({ ...acc, [`${curr[0]}`]: JSON.stringify(curr[1]) }),
      {}
    ),
  }),
  new CopyWebpackPlugin({
    patterns: [
      {
        from: "./src/manifest.json",
        // Set description and version in extension manifest.json from contents of package.json (so
        // that there's only one place to update).
        transform: function (content, _path) {
          return Buffer.from(
            JSON.stringify({
              ...JSON.parse(content.toString()),
              description: process.env.npm_package_description,
              version: process.env.npm_package_version,
            })
          );
        },
      },
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
// Intentionally disabling mangling and leaving source maps in both dev and prod builds because this
// is going to be source available anyway... might as well make debugging easier. This makes
// background.bundle.js be ~350KB instead of ~70KB, but that shouldn't  matter much given how rarely
// the JS is downloaded and parsed.
export const devtool = "inline-source-map";
export const optimization = {
  usedExports: true,
  minimize: true,
  minimizer: [
    new TerserPlugin({
      extractComments: false,
      terserOptions: {
        compress: {
          defaults: false,
          unused: true,
        },
        mangle: false,
        format: {
          comments: "all",
        },
      },
    }),
  ],
};
export const output = {
  filename: "[name].bundle.js",
  path: _resolve(__dirname, "dist"),
  chunkFormat: "commonjs",
  clean: true,
};

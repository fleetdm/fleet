require("es6-promise").polyfill();

const path = require("path");
const webpack = require("webpack");
const bourbon = require("node-bourbon").includePaths;
const HtmlWebpackPlugin = require("html-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const WebpackNotifierPlugin = require("webpack-notifier");
const ForkTsCheckerWebpackPlugin = require("fork-ts-checker-webpack-plugin");
const globImporter = require("node-sass-glob-importer");

const DEV_SOURCE_MAPS = "eval-source-map";

let plugins = [
  new ForkTsCheckerWebpackPlugin(),
  new HtmlWebpackPlugin({
    filename: "../frontend/templates/react.tmpl",
    inject: false,
    templateParameters: {
      isProduction: process.env.NODE_ENV === "production",
    },
    template: "frontend/templates/react.ejs",
  }),
  new WebpackNotifierPlugin({
    excludeWarnings: true,
  }),
];

if (process.env.NODE_ENV === "production") {
  plugins = plugins.concat([
    new webpack.DefinePlugin({
      "process.env": { NODE_ENV: JSON.stringify("production") },
    }),
    new MiniCssExtractPlugin({
      filename: "bundle-[contenthash].css",
    }),
  ]);
} else {
  // development
  plugins = plugins.concat([
    new MiniCssExtractPlugin({ filename: "bundle.css" }),
  ]);
}

const repo = __dirname;

const config = {
  mode: process.env.NODE_ENV,
  entry: {
    bundle: path.join(repo, "frontend/index.jsx"),
  },
  output: {
    path: path.join(repo, "assets/"),
    publicPath: "/assets/",
    filename: "[name].js",
  },
  devtool: process.env.NODE_ENV === "development" ? DEV_SOURCE_MAPS : false,
  plugins,
  optimization: {
    minimize: process.env.NODE_ENV === "production",
  },
  module: {
    // The following noParse suppresses the warning about sqlite-parser being a
    // pre-compiled JS file. See https://goo.gl/N4s6bB.
    noParse: /node_modules\/sqlite-parser\/dist\/sqlite-parser-min.js/,
    rules: [
      {
        test: /\.(pdf|png|gif|ico|jpg|svg|eot|otf|woff|woff2|ttf|mp4|webm)$/,
        type: "asset",
        generator: {
          filename: "[name]@[hash][ext]",
        },
      },
      {
        test: /\.(sh|ps1)$/,
        type: "asset/source",
      },
      {
        test: /(\.tsx?|\.jsx?)$/,
        exclude: /node_modules/,
        use: {
          loader: "esbuild-loader",
          options: {
            loader: "tsx", // Or 'ts' if you don't need tsx
            target: "es2016",
          },
        },
      },
      {
        test: /\.scss$/,
        exclude: /node_modules/,
        use: [
          {
            loader: MiniCssExtractPlugin.loader,
            options: {
              publicPath: "./",
            },
          },
          { loader: "css-loader" },
          { loader: "postcss-loader" },
          {
            loader: "sass-loader",
            options: {
              sourceMap: true,
              sassOptions: {
                includePaths: bourbon,
                importer: globImporter(),
              },
            },
          },
        ],
      },
      {
        test: /\.css$/,
        use: [
          {
            loader: MiniCssExtractPlugin.loader,
            options: {},
          },
          "css-loader",
          "postcss-loader",
        ],
      },
      {
        test: /\.jsx?$/,
        include: path.join(repo, "frontend"),
        use: { loader: "babel-loader", options: { cacheDirectory: true } },
      },
    ],
  },
  resolve: {
    extensions: [".tsx", ".ts", ".js", ".jsx", ".json"],
    modules: [path.resolve(path.join(repo, "./frontend")), "node_modules"],
    fallback: { path: require.resolve("path-browserify") },
  },
};

if (process.env.NODE_ENV === "production") {
  config.output.filename = "[name]-[contenthash].js";
}

module.exports = config;

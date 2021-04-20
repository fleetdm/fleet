require("es6-promise").polyfill();

const path = require("path");
const webpack = require("webpack");
const bourbon = require("node-bourbon").includePaths;
const HtmlWebpackPlugin = require("html-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const WebpackNotifierPlugin = require("webpack-notifier");
const HardSourceWebpackPlugin = require("hard-source-webpack-plugin");

const DEV_SOURCE_MAPS = "eval-source-map";

var plugins = [
  new webpack.NoEmitOnErrorsPlugin(),
  new HtmlWebpackPlugin({
    filename: "../frontend/templates/react.tmpl",
    inject: false,
    template: "frontend/templates/react.ejs",
  }),
  new WebpackNotifierPlugin({
    title: "Fleet",
    contentImage: path.resolve("./assets/images/kolide-logo.svg"),
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
      allChunks: false,
    }),
  ]);
} else {
  // development
  plugins = plugins.concat([
    new MiniCssExtractPlugin({ filename: "bundle.css", allChunks: false }),
    // Huge speedup on subsequent builds by caching modules
    new HardSourceWebpackPlugin({
      // Allow pruning anything over an hour old
      cachePrune: { maxAge: 60 * 60 * 1000 },
    }),
  ]);
}

var repo = __dirname;

var config = {
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
  plugins: plugins,
  optimization: {
    minimize: process.env.NODE_ENV === "production",
  },
  module: {
    // The following noParse suppresses the warning about sqlite-parser being a
    // pre-compiled JS file. See https://goo.gl/N4s6bB.
    noParse: /node_modules\/sqlite-parser\/dist\/sqlite-parser-min.js/,
    rules: [
      {
        test: /\.(png|gif)$/,
        use: { loader: "url-loader?name=[name]@[hash].[ext]&limit=6000" },
      },
      {
        test: /\.(pdf|ico|jpg|svg|eot|otf|woff|woff2|ttf|mp4|webm)$/,
        use: {
          loader: "file-loader",
          options: {
            name: "[name]@[hash].[ext]",
            useRelativePath: true,
          },
        },
      },
      {
        test: /\.tsx?$/,
        exclude: /node_modules/,
        use: { loader: "ts-loader" },
      },
      {
        test: /\.scss$/,
        exclude: /node_modules/,
        use: [
          {
            loader: MiniCssExtractPlugin.loader,
            options: {
              publicPath: "./",
              hmr: process.env.NODE_ENV === "development",
            },
          },
          { loader: "css-loader" },
          { loader: "postcss-loader" },
          {
            loader: "sass-loader",
            options: {
              sourceMap: true,
              includePaths: [bourbon],
            },
          },
          { loader: "import-glob-loader" },
        ],
      },
      {
        test: /\.css$/,
        use: [
          {
            loader: MiniCssExtractPlugin.loader,
            options: {
              hmr: process.env.NODE_ENV === "development",
            },
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
  },
};

if (process.env.NODE_ENV === "production") {
  config.output.filename = "[name]-[hash].js";
}

module.exports = config;

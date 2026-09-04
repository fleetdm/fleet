require("es6-promise").polyfill();

const path = require("path");
const webpack = require("webpack");
const bourbon = require("node-bourbon").includePaths;
const HtmlWebpackPlugin = require("html-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
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
  new webpack.DefinePlugin({
    featureFlags: {},
  }),
];

if (process.env.NODE_ENV === "production") {
  plugins = plugins.concat([
    new webpack.DefinePlugin({
      "process.env": { NODE_ENV: JSON.stringify("production") },
    }),
    new MiniCssExtractPlugin({
      filename: "bundle-[contenthash].css",
      // Async chunks need this too, or their CSS is emitted unhashed — which
      // the Go asset handler then serves with no-cache (see hashedAssetRe in
      // server/service/frontend.go). Keeping the bundle- prefix also keeps
      // these covered by the assets/bundle*.* gitignore rule.
      chunkFilename: "bundle-[contenthash].css",
    }),
  ]);
} else {
  // development
  plugins = plugins.concat([
    new MiniCssExtractPlugin({
      filename: "bundle.css",
      chunkFilename: "bundle-[name].css",
    }),
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
    rules: [
      {
        // Fleet-maintained app icons: always emitted as separate files, never
        // inlined as data URLs. There are thousands of them and a page renders at
        // most a screenful, so inlining any of them would put bytes in the
        // bundle that almost no page load needs.
        test: /\.png$/,
        include: path.join(
          repo,
          "frontend/pages/SoftwarePage/components/icons/png"
        ),
        type: "asset/resource",
        generator: {
          filename: "icons/[name]@[hash][ext]",
        },
      },
      {
        test: /\.(pdf|png|gif|ico|jpg|svg|eot|otf|woff|woff2|ttf|mp4|webm)$/,
        exclude: path.join(
          repo,
          "frontend/pages/SoftwarePage/components/icons/png"
        ),
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
                silenceDeprecations: [
                  "import",
                  "global-builtin",
                  "slash-div",
                  "color-functions",
                  "mixed-decls",
                  "legacy-js-api",
                ],
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
  performance: {
    hints: false,
  },
  resolve: {
    extensions: [".tsx", ".ts", ".js", ".jsx", ".json"],
    modules: [path.resolve(path.join(repo, "./frontend")), "node_modules"],
    fallback: { path: require.resolve("path-browserify") },
  },
};

if (process.env.NODE_ENV !== "production") {
  // Default splitChunks names shared chunks after their contents, so the names
  // shift as the sharing topology does — 404ing against go-bindata -debug's
  // asset list, which is frozen when generate-dev runs.
  config.optimization.splitChunks = {
    cacheGroups: {
      defaultVendors: {
        test: /[\\/]node_modules[\\/]/,
        name: "vendors",
        chunks: "async",
        priority: -10,
        reuseExistingChunk: true,
      },
      default: {
        name: "common",
        chunks: "async",
        minChunks: 2,
        priority: -20,
        reuseExistingChunk: true,
      },
    },
  };
}

if (process.env.NODE_ENV === "production") {
  config.output.filename = "[name]-[contenthash].js";
  // Route chunks need a content hash too, so the Go asset handler serves them
  // with a long-lived immutable Cache-Control (see hashedAssetRe in
  // server/service/frontend.go).
  config.output.chunkFilename = "[name]-[contenthash].js";
}

module.exports = config;

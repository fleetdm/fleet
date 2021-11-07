const path = require("path");
const bourbon = require("node-bourbon").includePaths;
const MiniCssExtractPlugin = require("mini-css-extract-plugin");

module.exports = {
  webpackFinal: async (config) => {
    config.module.rules.push({
      test: /\.scss$/,
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
    });

    config.plugins.push(new MiniCssExtractPlugin({ filename: '[name].css' }))
    config.resolve.modules.push(path.resolve(__dirname, '../frontend'));

    return config;
  },
  "stories": [
    "../frontend/components/**/*.stories.mdx",
    "../frontend/components/**/*.stories.@(js|jsx|ts|tsx)"
  ],
  "addons": [
    "@storybook/addon-links",
    "@storybook/addon-essentials"
  ]
}
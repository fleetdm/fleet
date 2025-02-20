const path = require("path");
const bourbon = require("node-bourbon").includePaths;
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const globImporter = require("node-sass-glob-importer");

import type { StorybookConfig } from "@storybook/react-webpack5";

const config: StorybookConfig = {
  webpackFinal: async (config) => {
    config.module?.rules?.push({
      test: /\.scss$/,
      use: [
        {
          loader: MiniCssExtractPlugin.loader,
          options: {
            publicPath: "./",
          },
        },
        {
          loader: "css-loader",
        },
        {
          loader: "postcss-loader",
        },
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
    });
    config.plugins?.push(
      new MiniCssExtractPlugin({
        filename: "[name].css",
      })
    );
    config.resolve?.modules?.push(path.resolve(__dirname, "../frontend"));
    return config;
  },
  stories: [
    "../frontend/components/**/*.stories.mdx",
    "../frontend/components/**/*.stories.@(js|jsx|ts|tsx)",
  ],
  addons: [
    "@storybook/addon-links",
    "@storybook/addon-essentials",
    "@storybook/addon-mdx-gfm",
    "@storybook/addon-a11y",
    "@storybook/test-runner",
    "@storybook/addon-designs",
    "@storybook/addon-webpack5-compiler-babel"
  ],
  typescript: {
    check: false,
    reactDocgen: "react-docgen-typescript",
    reactDocgenTypescriptOptions: {
      shouldExtractLiteralValuesFromEnum: true,
      propFilter: (prop) =>
        prop.parent ? !/node_modules/.test(prop.parent.fileName) : true,
      shouldRemoveUndefinedFromOptional: true,
    },
  },
  framework: {
    name: "@storybook/react-webpack5",
    options: {},
  },
  docs: {},
};

export default config;

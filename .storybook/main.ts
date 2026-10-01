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
    // Mirror webpack.config.js: extract .css alongside .scss so cascade
    // order matches import order. Without this, Storybook's default
    // style-loader injects .css (e.g. react-select/dist/react-select.css)
    // at runtime AFTER the extracted Fleet bundle, and any equal-specificity
    // rule from a vendor .css beats Fleet's override (e.g. dark-mode Dropdown
    // label color). Replace Storybook's default .css rule (rather than
    // pushing) so CSS isn't processed by both style-loader AND MiniCss.
    if (config.module?.rules) {
      config.module.rules = config.module.rules.filter((rule) => {
        if (rule && typeof rule === "object" && "test" in rule) {
          const t = rule.test;
          return !(t instanceof RegExp && t.test("x.css"));
        }
        return true;
      });
      config.module.rules.push({
        test: /\.css$/,
        use: [
          { loader: MiniCssExtractPlugin.loader, options: {} },
          "css-loader",
          "postcss-loader",
        ],
      });
    }
    config.plugins?.push(
      new MiniCssExtractPlugin({
        filename: "[name].css",
      })
    );
    config.resolve?.modules?.push(path.resolve(__dirname, "../frontend"));
    return config;
  },
  stories: [
    "../frontend/components/**/*.stories.@(js|jsx|ts|tsx)",
    "../frontend/pages/**/*.stories.@(js|jsx|ts|tsx)",
  ],
  addons: [
    "@storybook/addon-links",
    "@storybook/addon-essentials",
    "@storybook/addon-a11y",
    "@storybook/addon-designs",
    "@storybook/addon-webpack5-compiler-babel",
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

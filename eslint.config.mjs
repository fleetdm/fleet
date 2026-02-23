import path from "path";
import { fileURLToPath } from "url";
import js from "@eslint/js";
import globals from "globals";
import tseslint from "typescript-eslint";
import { fixupPluginRules } from "@eslint/compat";
import reactPlugin from "eslint-plugin-react";
import reactHooksPlugin from "eslint-plugin-react-hooks";
import importPlugin from "eslint-plugin-import";
import jestPlugin from "eslint-plugin-jest";
import jsxA11yPlugin from "eslint-plugin-jsx-a11y";
import prettierConfig from "eslint-config-prettier";
import storybookPlugin from "eslint-plugin-storybook";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export default tseslint.config(
  // Global ignores (replaces .eslintignore)
  {
    ignores: ["frontend/utilities/node-sql-parser"],
  },

  // Base JS recommended rules
  js.configs.recommended,

  // TypeScript recommended rules
  ...tseslint.configs.recommended,

  // Jest recommended config for test files
  {
    files: [
      "frontend/**/*.test.{js,jsx,ts,tsx}",
      "frontend/test/**/*.{js,jsx,ts,tsx}",
    ],
    ...jestPlugin.configs["flat/recommended"],
    rules: {
      ...jestPlugin.configs["flat/recommended"].rules,
      "jest/no-mocks-import": "off",
    },
  },

  // Storybook recommended config for story files
  ...storybookPlugin.configs["flat/recommended"],

  // Main frontend config
  {
    files: ["frontend/**/*.{js,jsx,ts,tsx}"],
    plugins: {
      react: fixupPluginRules(reactPlugin),
      "react-hooks": fixupPluginRules(reactHooksPlugin),
      import: fixupPluginRules(importPlugin),
      "jsx-a11y": fixupPluginRules(jsxA11yPlugin),
    },
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      globals: {
        ...globals.browser,
        ...globals.node,
        ...globals.mocha,
        ...globals.jest,
        expect: "readonly",
        describe: "readonly",
      },
      parserOptions: {
        ecmaFeatures: {
          jsx: true,
        },
      },
    },
    settings: {
      react: {
        version: "detect",
      },
      "import/resolver": {
        webpack: {
          config: path.join(__dirname, "webpack.config.js"),
        },
      },
    },
    rules: {
      // --- Core rules ---
      camelcase: "off",
      "consistent-return": "warn",
      "arrow-body-style": "off",
      "max-len": "off",
      "no-unused-expressions": "off",
      "no-console": "off",
      "space-before-function-paren": "off",
      "no-param-reassign": "off",
      "new-cap": "off",
      "linebreak-style": "off",
      "no-underscore-dangle": "off",
      // Disable base rule in favor of the TypeScript-aware version
      "no-use-before-define": "off",
      // Disable base rule in favor of the TypeScript-aware version
      "no-shadow": "off",

      // --- React rules ---
      "react/prefer-stateless-function": "off",
      "react/no-multi-comp": "off",
      "react/no-unused-prop-types": [
        "warn",
        {
          skipShapeProps: true,
        },
      ],
      "react/require-default-props": "off",
      "react/jsx-filename-extension": [
        "warn",
        {
          extensions: [".jsx", ".tsx"],
        },
      ],
      "react/prop-types": "off",

      // --- React Hooks rules ---
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",

      // --- Import rules ---
      "import/no-unresolved": [
        "error",
        {
          caseSensitive: false,
          ignore: [
            "^date-fns",
            "^date-fns-tz",
            "^use-debounce",
            "^@storybook/",
            "^@testing-library/jest-dom",
          ],
        },
      ],
      "import/no-named-as-default": "off",
      "import/no-named-as-default-member": "off",
      "import/extensions": "off",
      "import/no-extraneous-dependencies": "off",

      // --- JSX Accessibility rules ---
      "jsx-a11y/no-static-element-interactions": "off",
      "jsx-a11y/heading-has-content": "off",
      "jsx-a11y/anchor-has-content": "off",

      // --- TypeScript rules ---
      "@typescript-eslint/no-use-before-define": ["error"],
      "@typescript-eslint/explicit-module-boundary-types": "off",
      "@typescript-eslint/ban-ts-comment": "off",
      "@typescript-eslint/no-shadow": "error",
      "@typescript-eslint/no-explicit-any": "warn",
      "@typescript-eslint/no-unused-vars": [
        "error",
        {
          argsIgnorePattern: "^_",
          varsIgnorePattern: "^_",
          caughtErrorsIgnorePattern: "^_",
          caughtErrors: "none",
          ignoreRestSiblings: true,
        },
      ],
      "@typescript-eslint/no-unused-expressions": "off",
      "@typescript-eslint/no-require-imports": "off",
    },
  },

  // Prettier config must be last to override all formatting rules
  prettierConfig
);

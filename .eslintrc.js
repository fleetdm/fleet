var path = require("path");
module.exports = {
  extends: [
    "airbnb",
    "plugin:jest/recommended",
    "plugin:react-hooks/recommended",
    "plugin:@typescript-eslint/recommended",
    "plugin:prettier/recommended",
    "plugin:storybook/recommended",
  ],
  parser: "@typescript-eslint/parser",
  plugins: ["jest", "react", "@typescript-eslint"],
  env: {
    node: true,
    mocha: true,
    browser: true,
    "jest/globals": true,
  },
  globals: {
    expect: false,
    describe: false,
  },
  rules: {
    camelcase: "off",
    "consistent-return": 1,
    "arrow-body-style": 0,
    "max-len": 0,
    "no-unused-expressions": 0,
    "no-console": 0,
    "space-before-function-paren": 0,
    "react/prefer-stateless-function": 0,
    "react/no-multi-comp": 0,
    "react/no-unused-prop-types": [
      1,
      {
        customValidators: [],
        skipShapeProps: true,
      },
    ],
    "react/require-default-props": 0,
    // TODO set default props and enable this check
    "react/jsx-filename-extension": [
      1,
      {
        extensions: [".jsx", ".tsx"],
      },
    ],
    "react-hooks/exhaustive-deps": 1,
    "no-param-reassign": 0,
    "new-cap": 0,
    "import/no-unresolved": [
      2,
      {
        caseSensitive: false,
      },
    ],
    "linebreak-style": 0,
    "import/no-named-as-default": "off",
    "import/no-named-as-default-member": "off",
    "import/extensions": 0,
    "import/no-extraneous-dependencies": 0,
    "no-underscore-dangle": 0,
    "jsx-a11y/no-static-element-interactions": "off",
    // note you must disable the base rule as it can report incorrect errors. more info here:
    // https://github.com/typescript-eslint/typescript-eslint/blob/master/packages/eslint-plugin/docs/rules/no-use-before-define.md
    "no-use-before-define": "off",
    "@typescript-eslint/no-use-before-define": ["error"],
    // turn off and override to not run this on js and jsx files. More info here:
    // https://github.com/typescript-eslint/typescript-eslint/blob/master/packages/eslint-plugin/docs/rules/explicit-module-boundary-types.md#configuring-in-a-mixed-jsts-codebase
    "@typescript-eslint/explicit-module-boundary-types": "off",
    // Most of the js modules written by us need to be rewritten into TS. Until then
    // we use ts-ignore comment to ignore the error TS gives us from not having those modules
    // declared (TS7016). This is done on purpose as there is not time to rewrite everything in TS.
    "@typescript-eslint/ban-ts-comment": "off",
    "no-shadow": "off",
    // replaced by ts-eslint rule below
    "@typescript-eslint/no-shadow": "error",
    // There is a bug with these rules in our version of jsx-a11y plugin (5.1.1)
    // To upgrade our version of the plugin we would need to make more changes
    // with eslint-config-airbnb, so we will just turn off for now.
    "jsx-a11y/heading-has-content": "off",
    "jsx-a11y/anchor-has-content": "off",
  },
  overrides: [],
  settings: {
    "import/resolver": {
      webpack: {
        config: path.join(__dirname, "webpack.config.js"),
      },
    },
  },
};

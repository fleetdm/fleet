var path = require("path");
module.exports = {
  extends: ["plugin:fleet-lint/recommended"],
  rules: {},
  settings: {
    "import/resolver": {
      webpack: {
        config: path.join(__dirname, "webpack.config.js"),
      },
    },
  },
};

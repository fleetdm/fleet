var path = require('path');

module.exports = {
  extends: [
    'airbnb',
    'plugin:jest/recommended',
  ],
  parser: 'babel-eslint',
  plugins: [
    'jest',
    'react',
  ],
  env: {
    'node': true,
    'mocha': true,
    'browser': true,
    'jest/globals': true,
  },
  globals: {
    'expect': false,
    'describe': false,
  },
  rules: {
    'consistent-return': 1,
    'arrow-body-style': 0,
    'max-len': 0,
    'no-use-before-define': [2, 'nofunc'],
    'no-unused-expressions': 0,
    'no-console': 0,
    'space-before-function-paren': 0,
    'react/prefer-stateless-function': 0,
    'react/no-multi-comp': 0,
    'react/no-unused-prop-types': [1, { 'customValidators': [], skipShapeProps: true }],
    'react/require-default-props': 0, // TODO set default props and enable this check
    'no-param-reassign': 0,
    'new-cap': 0,
    'import/no-unresolved': 2,
    'linebreak-style': 0,
    'import/no-named-as-default': 'off',
    'import/no-named-as-default-member': 'off',
    'import/extensions': 0,
    'import/no-extraneous-dependencies': 0,
    'no-underscore-dangle': 0,
    'jsx-a11y/no-static-element-interactions': 'off'
  },
  settings: {
    'import/resolver': {
      webpack: {
        config: path.join(__dirname, 'webpack.config.js')
      }
    }
  }
}

var path = require('path');

module.exports = {
  extends: 'airbnb',
  parser: 'babel-eslint',
  plugins: [
    'react'
  ],
  env: {
    'node': true,
    'mocha': true
  },
  globals: {
    'expect': false,
    'describe': false
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
    'no-param-reassign': 0,
    'new-cap': 0,
    'import/no-unresolved': 'error',
    'linebreak-style': 0,
    'import/no-named-as-default': 'off',
    'import/no-named-as-default-member': 'off',
    'import/extensions': 0,
    'import/no-extraneous-dependencies': 0,
    'import/no-unresolved': 0,
    'no-underscore-dangle': 0
  },
  settings: {
    'import/resolver': {
      webpack: {
        config: path.join(__dirname, 'webpack.config.js')
      }
    }
  }
}

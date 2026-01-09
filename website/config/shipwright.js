/**
 * Shipwright configuration
 *
 * Modern asset pipeline powered by Rsbuild.
 *
 * @see https://github.com/sailshq/sails-hook-shipwright
 */

const { pluginLess } = require('@rsbuild/plugin-less');

module.exports.shipwright = {
  js: {
    // Bundle all JS files in order (matches tasks/pipeline.js)
    entry: [
      'js/cloud.setup.js',
      'js/components/**/*.js',
      'js/utilities/**/*.js',
      'js/pages/**/*.js'
    ],
    // Dependencies loaded as separate scripts before bundle
    inject: [
      'dependencies/sails.io.js',
      'dependencies/lodash.js',
      'dependencies/jquery.min.js',
      'dependencies/vue.js',
      'dependencies/vue-router.js',
      'dependencies/**/*.js'
    ]
  },
  build: {
    plugins: [pluginLess()]
  }
};

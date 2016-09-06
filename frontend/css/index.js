require('normalize.css');
require('./global.css');

/**
 * Components.
 * Include all css files just if you need
 * to hot reload it. And make sure that you
 * use `webpack.optimize.DedupePlugin`
 */
require('../components/app/styles.css');

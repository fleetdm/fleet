require('normalize.css');
require('#css/global');

/**
 * Components.
 * Include all css files just if you need
 * to hot reload it. And make sure that you
 * use `webpack.optimize.DedupePlugin`
 */
require('#app/components/app/styles');

/**
 * Module dependencies
 */

var util = require('util');
var path = require('path');
var _ = require('@sailshq/lodash');
// var generateFile = require('./node_modules/sails-generate/builtins/file');

/**
 * @eashaw/sails-generate-landing-page
 *
 * Usage:
 * `sails generate landing-page`
 *
 * @description Generates a landing-page.
 * @docs https://sailsjs.com/docs/concepts/extending-sails/generators/custom-generators
 */

module.exports = {

  templatesDirectory: path.resolve(__dirname,'./templates'),

  /**
   * Scope:
   * ----------------------------------------------------
   * @option {Array} args   [command-line arguments]
   * ----------------------------------------------------
   * @property {String} relPath
   * @property {String} stem
   * @property {String} newActionSlug
   *
   * @property {String} newViewRelPath
   * @property {String} newActionRelPath
   * @property {String} newStylesheetRelPath
   * @property {String} newPageScriptRelPath
   */

  before: function (scope, exits) {
    if (!scope.args[0]) {
      return exits.error(
        'Please specify the base name or path for the new landing page.\n'+
        '(relative from the `views/pages/imagine/` folder;\n'+
        ' e.g. `osquer-managmenet`)'
      );
    }

    // e.g. `dashboard/activity-summary`
    scope.relPath = scope.args[0];

    // Check if it has a file extension and, if so, reject it.
    if (path.extname(scope.relPath)) {
      return exits.error('Please specify the path for the new page, excluding the filename suffix (i.e. no ".ejs")');
    }

    // Trim any whitespace from both sides.
    scope.relPath = _.trim(scope.relPath);

    // Replace backslashes with proper slashes.
    // (This is crucial for Windows compatibility.)
    scope.relPath = scope.relPath.replace(/\\/g, '/');

    // Check that it does not have any trailing slashes.
    if (scope.relPath.match(/\/$/)) {
      return exits.error('Please specify the path for the new page. (No trailing slash please.)');
    }

    // Check that it does not begin with a slash or a dot dot slash.
    // (a single dot+slash is ok, since you might be using tab-completion in the terminal)
    if (scope.relPath.match(/^\.\.+\//) || scope.relPath.match(/^\//)) {
      return exits.error('No need for dots and leading slashes and things. Please specify something like: `dashboard/activity-summary`');
    }

    // Make sure the relative path is not within "pages/", "views/", "controllers/",
    // "assets/", "js/", "styles/", or anything else like that.  If it is, it's probably
    // an accident.  And if it's not an accident, it's still super confusing.
    if (scope.relPath.match(/^(pages\/|views\/|controllers\/|api\/|assets\/|js\/|styles\/)/i)) {
      return exits.error('Please specify *just* the relative path for the new page, excluding prefixes like "pages/", "views/", or "controllers/".  Those will be attached for you automatically-- you just need to include the last bit; e.g. `dashboard/activity-summary` or  `internal/admin-activity-log`');
    }

    // Gracefully ignore double-slashes.
    scope.relPath = scope.relPath.replace(/\/\/+/, '/');

    // Gracefully ignore leading "./", if present.
    scope.relPath = scope.relPath.replace(/^[\.\/]+/, '');

    // Make sure all parent sub-folders are kebab-cased and don't contain any
    // uppercase or non-alphanumeric characters (except dashes are ok, of course).
    var parentSubFoldersString = path.dirname(scope.relPath);
    var arrayOfParentSubFolders = ['imagine'];

    // Tease out the "stem".
    // (e.g. `activity-summary`)
    var stem = path.basename(scope.relPath);

    // Then kebab-case it, if it isn't already.
    // (e.g. `activitySummary` becomes `activity-summary`)
    stem = _.kebabCase(stem);

    // Make sure it doesn't start with `view`.
    // (e.g. NOT `view-activity-summary`)
    if (stem.match(/^view-/)) {
      return exits.error('No need to prefix with "view-" when generating a page.  Instead, just leave that part out.  (It\'ll be added automatically where needed.)');
    }

    // Check that the stem doesn't still contain any uppercase or non-alphanumeric
    // characters.  (Except dashes are ok, of course.)
    if (stem.match(/[^a-z0-9\-]/) || stem !== _.deburr(stem)) {
      return exits.error('Please stick to alphanumeric characters and dashes.');
    }


    // ◊  (Now then…)
    scope.stem = stem;
    scope.newActionSlug = path.join(arrayOfParentSubFolders.join('/'), 'view-'+stem);
    scope.newActionRelPath = path.join('api/controllers/', scope.newActionSlug+'.js');
    scope.newViewRelPath = path.join('views/pages/imagine/', scope.relPath+'.ejs');
    scope.newStylesheetRelPath = path.join('assets/styles/pages/imagine/', scope.relPath+'.less');
    scope.newPageScriptRelPath = path.join('assets/js/pages/imagine/', scope.relPath+'.page.js');

    // Set up underlying "action" generator.
    scope.actions2 = true;
    scope.args = [ scope.newActionSlug ];

    // Disable the "Created a new …!" output so we can use our own instead.
    scope.suppressFinalLog = true;

    return exits.success();
  },

  after: function (scope, done){
    console.log();
    console.log('Successfully generated:');
    console.log(' •-',scope.newViewRelPath);
    console.log(' •-',scope.newActionRelPath);
    console.log(' •-',scope.newStylesheetRelPath);
    console.log(' •-',scope.newPageScriptRelPath);
    console.log();
    console.log('A few reminders:');
    console.log(' (1)  These files were generated assuming your Sails app is using');
    console.log('      Vue.js as its front-end framework.  (If you\'re unsure,');
    console.log('      head over to https://sailsjs.com/support)');
    console.log();
    console.log(' (2)  You\'ll need to manually add a route for this new page\'s');
    console.log('      action in your `config/routes.js` file; e.g.');
    console.log('          \'GET /imagine/'+scope.relPath+'\': { action: \''+(
      scope.newActionSlug.replace(/\\/g,'/')//« because Windows
    )+'\' },');
    console.log();
    console.log(' (3)  You\'ll need to manually import the new LESS stylesheet');
    console.log('      from your `assets/styles/importer.less` file; e.g.');
    console.log('          @import \''+(
      path.join('pages/imagine/', scope.relPath+'.less').replace(/\\/g,'/')//« because Windows
    )+'\';');
    console.log();
    console.log(' (4)  Last but not least, since some of the above are backend changes,');
    console.log('      don\'t forget to re-lift the server before testing!');
    console.log();

    return done();
  },

  targets: {
    './': ['action'],// << Use underlying default generator
    './:newViewRelPath': { template: 'page.ejs.template' },
    './:newStylesheetRelPath': { template: 'stylesheet.less.template' },
    './:newPageScriptRelPath': { template: 'page-script.page.js.template' }
  }

};

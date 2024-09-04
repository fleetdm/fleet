/**
 * Module dependencies
 */

var path = require('path');
var _ = require('@sailshq/lodash');
var generateFile = require('../../node_modules/sails-generate/lib/builtins/file');

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
        ' e.g. `osquery-managmenet`)'
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
    if (scope.relPath.match(/^(pages\/|views\/|controllers\/|api\/|assets\/|js\/|styles\/|imagine\/)/i)) {
      return exits.error('Please specify *just* the name of the new page, excluding prefixes like "pages/", "views/", or "controllers/".  Those will be attached for you automatically-- you just need to include the last bit; e.g. `security-compliance` or  `vulnerability-management`');
    }

    // Gracefully ignore double-slashes.
    scope.relPath = scope.relPath.replace(/\/\/+/, '/');

    // Gracefully ignore leading "./", if present.
    scope.relPath = scope.relPath.replace(/^[\.\/]+/, '');

    // Make sure all parent sub-folders are kebab-cased and don't contain any
    // uppercase or non-alphanumeric characters (except dashes are ok, of course).
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
    console.log('Successfully generated landing page:');
    console.log(' •-',scope.newViewRelPath);
    console.log(' •-',scope.newActionRelPath);
    console.log(' •-',scope.newStylesheetRelPath);
    console.log(' •-',scope.newPageScriptRelPath);
    console.log();
    console.log('A few reminders:');
    console.log(' (1)  These files were generated with lorem ipsum and ');
    console.log('      placeholder images. You\'ll need to edit the .ejs');
    console.log('      file to add the real content to the page.');
    console.log();
    console.log(' (2)  You\'ll need to manually add this route for the new page\'s');
    console.log('      action in the "Imagine" section of the `website/config/routes.js` file.');
    console.log('      Be sure to replace the TODOs with the page\'s real meta description and title.');
    console.log();
    console.log('\t\'GET /imagine/'+scope.relPath+'\': {\n\t\taction: \''+(scope.newActionSlug.replace(/\\/g,'/'))+'\',\n\t\tlocals: {\n\t\t\tpageTitleForMeta: \'TODO\',\n\t\t\tpageDescriptionForMeta: \'TODO\',\n\t\t}\n\t},');
    console.log();
    console.log(' (3)  You\'ll need to manually import the new LESS stylesheet');
    console.log('      from your `assets/styles/importer.less` file; Add this line');
    console.log('      to the same section as the other pages in the imagine folder:');
    console.log();
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
    './:newActionRelPath': {
      exec: function(scope, done){
        return generateFile({
          rootPath: scope.rootPath,
          force: scope.force,
          contents:
`module.exports = {


  friendlyName: 'View ${scope.stem.replace(/\-/gim, ' ')}',


  description: 'Display "${_.capitalize(scope.stem.replace(/\-/gim, ' '))}" page.',


  exits: {

    success: {
      viewTemplatePath: '${scope.newViewRelPath}'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = sails.config.builtStaticContent.testimonials;
    // Respond with view.
    return {
      testimonialsForScrollableTweets,
    };

  }


};`
        }, done);
      }
    },
    './:newViewRelPath': {
      exec: function(scope, done){
        return generateFile({
          rootPath: scope.rootPath,
          force: scope.force,
          contents:
`<div id="${scope.stem}" v-cloak>

  <div purpose="hero-background" class="container-lg d-flex justify-content-center">

    <div purpose="hero-container" class="d-flex flex-column align-items-center">
      <div purpose="hero-text">
        <h4 class="mb-2">A magni, recusandae qui omnis in quod?</h4>
        <h1>${_.capitalize(scope.stem.replace(/\-/gim, ' '))}</h1>
        <p>Vitae architecto reiciendis in temporibus consequatur doloremque reprehenderit perferendis? Eaque quod voluptates earum corporis, quo labore reprehenderit libero sint.</p>
        <div purpose="button-row" class="d-flex flex-sm-row flex-column justify-content-center align-items-center">
          <a purpose="cta-button" href="/register">Start now</a>
          <animated-arrow-button href="/contact">Talk to us</animated-arrow-button>
        </div>
      </div>
    </div>

  </div>

  <div purpose="page-container" class="container-lg">
    <div purpose="feature" class="d-flex flex-md-row flex-column-reverse justify-content-between mx-auto align-items-center">
      <div class="d-flex flex-column">
        <h3>Culpa ab maiores alias ut, non.</h3>
        <p>Do vero et, voluptatem fugiat consequatur? totam ut dolores ut velit! Sapiente et quasi et voluptas et doloribus quaerat minima saepe enim?</p>
        <div purpose="checklist" class="flex-column d-flex">
          <p>Necessitatibus qui harum non dolore a quis!</p>
          <p>Mollitia ipsum dicta hic quidem aut debitis qui iusto.</p>
          <p>Esse dolor animi nostrum quidem expedita.</p>
        </div>
      </div>
      <div purpose="feature-image" class="ml-md-5">
        <img alt="${scope.stem} feature image" src="https://my.feralgoblin.com/cass/300/300?text=${scope.stem.replace(/\-/gim, ' ')}+image+1">
      </div>
    </div>

    <div purpose="feature" class="d-flex flex-md-row flex-column justify-content-between mx-auto align-items-center">
      <div purpose="feature-image" class="mr-md-5">
        <img alt="${scope.stem} feature image" src="https://my.feralgoblin.com/cass/300/300?text=${scope.stem.replace(/\-/gim, ' ')}+image+2">
      </div>
      <div class="d-flex flex-column">
        <h3>Doloribus natus nihil alias earum, eius ducimus.</h3>
        <p>Laboriosam iusto magnam suscipit quasi ut quod, non, exercitationem. Enim aut consectetur repellat nihil pariatur? blanditiis.</p>
        <div purpose="checklist" class="flex-column d-flex">
          <p>Necessitatibus qui harum non dolore a quis!</p>
          <p>Mollitia ipsum dicta hic quidem aut debitis qui iusto.</p>
          <p>Esse dolor animi nostrum quidem expedita.</p>
        </div>
      </div>
    </div>

    <div purpose="feature" class="d-flex flex-md-row flex-column-reverse justify-content-between mx-auto align-items-center">
      <div class="d-flex flex-column">
        <h3>Ex, illum voluptates impedit nulla, et sunt cupiditate quas illo mollitia pariatur?? Tempora et delectus ut atque vel eaque optio, inventore.</h3>
        <p>Similique ea provident ducimus nulla ea debitis sequi nihil.</p>
        <div purpose="checklist" class="flex-column d-flex">
          <p>Necessitatibus qui harum non dolore a quis!</p>
          <p>Mollitia ipsum dicta hic quidem aut debitis qui iusto.</p>
          <p>Esse dolor animi nostrum quidem expedita.</p>
        </div>
      </div>
      <div purpose="feature-image" class="ml-md-5">
        <img alt="${scope.stem} feature image" src="https://my.feralgoblin.com/cass/300/300?text=${scope.stem.replace(/\-/gim, ' ')}+image+3">
      </div>
    </div>

    <div purpose="button-row" style="margin-top: 60px;" class="d-flex flex-sm-row flex-column justify-content-center align-items-center mx-auto">
      <a purpose="cta-button" href="/register">Start now</a>
      <animated-arrow-button href="/contact">Talk to us</animated-arrow-button>
    </div>

  </div>

  <div purpose="bottom-gradient">
    <div purpose="tweets-container" class="container-fluid px-md-0 pb-0 d-flex flex-column justify-content-center">
      <div purpose="section-heading" style="max-width: 720px" class="mx-auto text-center">
        <h4>Don’t know osquery?</h4>
        <h2>Dedicated support from osquery experts</h2>
        <p>Osquery is the open-source agent that powers Fleet. And we have the most osquery experts around. We’ll help you realize the potential of this tool for your organization.</p>
      </div>
    </div>

    <scrollable-tweets :testimonials="testimonialsForScrollableTweets"></scrollable-tweets>
    <div purpose="page-container" class="pb-0 container">

      <div purpose="bottom-cta" class="text-center">
        <h4>Open-source device management</h4>
        <h1>Lighter than air</h1>
        <div purpose="button-row" style="margin-top: 60px;" class="d-flex flex-sm-row flex-column justify-content-center align-items-center mx-auto">
          <a purpose="cta-button" href="/register">Start now</a>
          <animated-arrow-button href="/contact">Talk to us</animated-arrow-button>
        </div>
      </div>
    </div>
  </div>
  <div class="d-flex flex-column" purpose="bottom-cloud-city-banner">
    <img alt="A glass city floating on top of fluffy white clouds" class="d-none d-lg-flex" src="/images/homepage-cloud-city-banner-lg-1600x375@2x.png">
    <img alt="A glass city floating on top of fluffy white clouds" class="d-none d-md-flex d-lg-none" src="/images/homepage-cloud-city-banner-md-990x375@2x.png">
    <img alt="A glass city floating on top of fluffy white clouds" class="d-flex d-md-none" src="/images/homepage-cloud-city-banner-sm-375x168@2x.png">
  </div>
</div>
<%- /* Expose server-rendered data as window.SAILS_LOCALS :: */ exposeLocalsToBrowser() %>\n`
        }, done);
      }
    },
    './:newStylesheetRelPath': { template: 'stylesheet.less.template' },
    './:newPageScriptRelPath': { template: 'page-script.page.js.template' }
  }

};

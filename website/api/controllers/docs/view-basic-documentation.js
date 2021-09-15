module.exports = {


  friendlyName: 'View basic documentation',


  description: 'Display "Basic documentation" page.',


  urlWildcardSuffix: 'pageUrlSuffix',


  inputs: {
    pageUrlSuffix : {
      description: 'The relative path to the doc page from within this route.  (i.e. the URL wildcard suffix)',
      example: 'using-fleet/supported-browsers',
      type: 'string',
      defaultsTo: ''
    }
  },


  exits: {
    success: { viewTemplatePath: 'pages/docs/basic-documentation' },
    badConfig: { responseType: 'badConfig' },
    notFound: { responseType: 'notFound' },
    redirect: { responseType: 'redirect' },
  },


  fn: async function ({pageUrlSuffix}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }

    let SECTION_URL_PREFIX = '/docs';

    // Serve appropriate page content.
    // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js
    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, {
      url: _.trimRight(SECTION_URL_PREFIX + '/' + _.trim(pageUrlSuffix, '/'), '/')
    });
    // console.log('pageUrlSuffix:',pageUrlSuffix);
    // console.log('SECTION_URL_PREFIX + "/" + _.trim(pageUrlSuffix, "/"):',SECTION_URL_PREFIX + '/' + _.trim(pageUrlSuffix, '/'));
    // console.log('thisPage:',thisPage);

    // if the url doesn't match any existing page, or the page it matches doesn't match the url provided
    // then it might have extra slashes or uppercase characters (e.g. fleetdm.com/docs///usiNG-fleet////)
    // Note: because this also handles the docs landing page and a `pageUrlSuffix` might not have
    // been provided, we won't be rechecking the url if thisPage.url is '/docs'
    if (!thisPage || (thisPage.url !== '/docs/'+pageUrlSuffix && thisPage.url !== '/docs')) {
      // creating a regex to match instances of multiple slashes
      let multipleSlashesRegex = /\/{2,}/g;
      // Creating a lowercase double-slashless url to search with
      let modifiedPageUrlSuffix = pageUrlSuffix.toLowerCase().replace(multipleSlashesRegex, '/');
      // Finding the appropriate page content using the modified url.
      let revisedPage = _.find(sails.config.builtStaticContent.markdownPages, {
        url: _.trimRight(SECTION_URL_PREFIX + '/' + _.trim(modifiedPageUrlSuffix, '/'), '/')
      });
      // If we matched a page with the modified url, then redirect to that.
      if(revisedPage && revisedPage.url) {
        throw {redirect: revisedPage.url};
      } else {
        // otherwise, throw a 404 error.
        throw 'notFound';
      }
    }

    if (!thisPage) {
      throw 'notFound';
    }


    // Respond with view.
    return {
      path: require('path'),
      thisPage: thisPage,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath
    };

  }


};

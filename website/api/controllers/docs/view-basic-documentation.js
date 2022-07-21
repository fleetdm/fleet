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
    // Note that this action also serves the '/docs' landing page, as well as individual doc pages.
    // 
    // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js
    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, { url: SECTION_URL_PREFIX + '/' + pageUrlSuffix });
    if (!thisPage) {// If there's no matching page, try a revised version of the URL suffix that's lowercase, with internal slashes deduped, and any trailing slash or whitespace trimmed
      let revisedPageUrlSuffix = pageUrlSuffix.toLowerCase().replace(/\/+/g, '/').replace(/\/+\s*$/,'');
      thisPage = _.find(sails.config.builtStaticContent.markdownPages, { url: SECTION_URL_PREFIX + '/' + revisedPageUrlSuffix });
      if (thisPage) {// If we matched a page with the revised suffix, then redirect to that rather than rendering it, so the URL gets cleaned up.
        throw {redirect: thisPage.url};
      } else {// If no page could be found even with the revised suffix, then throw a 404 error.
        throw 'notFound';
      }
    }

    // Setting the meta title for this page.
    let pageTitleForMeta;
    if(thisPage.title === 'Readme.md') {
      // If thisPage.title is 'Readme.md', we're on the docs landing page and we'll follow the title format of the other top level pages.
      pageTitleForMeta = 'Documentation | Fleet for osquery';
    } else {
      // Otherwise we'll use the page title provided and format it accordingly.
      pageTitleForMeta = thisPage.title + ' | Fleet documentation';
    }
    // Setting the meta description for this page if one was provided, otherwise setting a generic description.
    let pageDescriptionForMeta = thisPage.meta.description ? thisPage.meta.description : 'Documentation for Fleet for osquery.';

    // Respond with view.
    return {
      path: require('path'),
      thisPage: thisPage,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath,
      pageTitleForMeta,
      pageDescriptionForMeta,
    };

  }


};

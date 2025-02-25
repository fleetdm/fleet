module.exports = {


  friendlyName: 'View basic handbook',


  description: 'Display "Basic handbook" page.',


  urlWildcardSuffix: 'pageUrlSuffix',


  inputs: {
    pageUrlSuffix : {
      description: 'The relative path to the doc page from within this route.  (i.e. the URL wildcard suffix)',
      example: 'handbook/release-process',
      type: 'string',
      defaultsTo: ''
    }
  },


  exits: {
    success: { viewTemplatePath: 'pages/handbook/basic-handbook' },
    badConfig: { responseType: 'badConfig' },
    notFound: { responseType: 'notFound' },
    redirect: { responseType: 'redirect' },
  },


  fn: async function ({pageUrlSuffix}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }

    let SECTION_URL_PREFIX = '/handbook';

    // Lookup appropriate page content, tolerating (but redirecting to fix) any unexpected capitalization or slashes.
    // Note that this action serves the '/handbook' landing page, as well as individual content pages therein.
    // (See also view-basic-documentation.js for the implementation this is based on)
    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, {
      url: (
        !pageUrlSuffix? SECTION_URL_PREFIX// « landing page (guaranteed to exist)
        : SECTION_URL_PREFIX + '/' + pageUrlSuffix// « individual content page
      )
    });
    if (!thisPage) {// If there's no EXACTLY matching content page, try a revised version of the URL suffix that's lowercase, with all slashes deduped, and any leading or trailing slash removed (leading slashes are only possible if this is a regex, rather than "/*" route)
      let revisedPageUrlSuffix = pageUrlSuffix.toLowerCase().replace(/\/+/g, '/').replace(/^\/+/,'').replace(/\/+$/,'');
      thisPage = _.find(sails.config.builtStaticContent.markdownPages, { url: SECTION_URL_PREFIX + '/' + revisedPageUrlSuffix });
      if (thisPage) {// If we matched a page with the revised suffix, then redirect to that rather than rendering it, so the URL gets cleaned up.
        throw {redirect: thisPage.url};
      } else {// If no page could be found even with the revised suffix, then throw a 404 error.
        throw 'notFound';
      }
    }


    // Respond with view.
    return {
      path: require('path'),
      thisPage: thisPage,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath,
      pageTitleForMeta: (
        thisPage.title !== 'Readme.md' ? thisPage.title + ' | Fleet handbook'// « custom meta title for this page, if provided in markdown
        : 'Handbook' // « otherwise we're on the landing page for this section of the site, so we'll follow the title format of other top-level pages
      ),
      pageDescriptionForMeta: (
        thisPage.meta.description ? thisPage.meta.description // « custom meta description for this page, if provided in markdown
        : 'View the Fleet handbook.'// « otherwise use a generic description
      ),
      rituals: sails.config.builtStaticContent.rituals,
      openPositions: sails.config.builtStaticContent.openPositions,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

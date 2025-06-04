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

    // Lookup appropriate page content, tolerating (but redirecting to fix) any unexpected capitalization or slashes.
    // Note that this action serves the '/docs' landing page, as well as individual doc pages.
    // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js
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

    let showSwagForm = false;
    // Due to shipping costs, we'll check the requesting user's cf-ipcountry to see if they're in the US, and their cf-iplongitude header to see if they're in the contiguous US.
    if(sails.config.environment === 'production') {
      // Log a warning if the cloudflare headers we use are missing in production.
      if(!this.req.get('cf-ipcountry') || !this.req.get('cf-iplongitude')) {
        sails.log.warn('When a user visted the docs, the Cloudflare header we use to determine if they are visiting from the contiguous United States is missing.');
      }
    }
    if(this.req.get('cf-ipcountry') === 'US' && this.req.get('cf-iplongitude') > -125) {
      showSwagForm = true;
    }

    // Respond with view.
    return {
      path: require('path'),
      thisPage: thisPage,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath,
      pageTitleForMeta: (
        thisPage.title !== 'Readme.md' ? thisPage.title + ' | Fleet documentation'// « custom meta title for this page, if provided in markdown
        : 'Documentation' // « otherwise we're on the landing page for this section of the site, so we'll follow the title format of other top-level pages
      ),
      pageDescriptionForMeta: (
        thisPage.meta.description ? thisPage.meta.description // « custom meta description for this page, if provided in markdown
        : 'Documentation for Fleet for osquery.'// « otherwise use the generic description
      ),
      showSwagForm,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

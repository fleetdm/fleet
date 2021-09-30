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

    // Setting a flag if the pageUrlSuffix doesn't match any existing page, or if the page it matches doesn't exactly match the pageUrlSuffix provided
    // Note: because this also handles the docs landing page and a pageUrlSuffix might not have provided, we set this flag to false if the url is just '/docs'
    let needsRedirectMaybe = (!thisPage || (thisPage.url !== '/docs/'+pageUrlSuffix && thisPage.url !== '/docs'));

    if (needsRedirectMaybe) {
      // Creating a lower case, repeating-slashless pageUrlSuffix
      let multipleSlashesRegex = /\/{2,}/g;
      let modifiedPageUrlSuffix = pageUrlSuffix.toLowerCase().replace(multipleSlashesRegex, '/');
      // Finding the appropriate page content using the modified pageUrlSuffix.
      let revisedPage = _.find(sails.config.builtStaticContent.markdownPages, {
        url: _.trimRight(SECTION_URL_PREFIX + '/' + _.trim(modifiedPageUrlSuffix, '/'), '/')
      });
      if(revisedPage) {
        // If we matched a page with the modified pageUrlSuffix, then redirect to that.
        throw {redirect: revisedPage.url};
      } else {
        // If no page was found, throw a 404 error.
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

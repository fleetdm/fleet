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

    // Serve appropriate page content.
    // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js
    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, {
      url: _.trimRight(SECTION_URL_PREFIX + '/' + _.trim(pageUrlSuffix, '/'), '/')
    });

    // Setting a flag if the pageUrlSuffix doesn't match any existing page, or if the page it matches doesn't exactly match the pageUrlSuffix provided
    // Note: because this also handles fleetdm.com/handbook and a pageUrlSuffix might not have provided, we set this flag to false if the url is just '/handbook'
    let needsRedirectMaybe = (!thisPage || (thisPage.url !== '/handbook/'+pageUrlSuffix && thisPage.url !== '/handbook'));
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
      // If thisPage.title is 'Readme.md', we're on the handbook landing page and we'll follow the title format of the other top level pages.
      pageTitleForMeta = 'Handbook | Fleet for osquery';
    } else {
      // Otherwise we'll use the page title provided and format it accordingly.
      pageTitleForMeta = thisPage.title + ' | Fleet handbook';
    }
    // Setting the meta description for this page if one was provided in the markdown, otherwise setting a generic description.
    let pageDescriptionForMeta = thisPage.meta.description ? thisPage.meta.description : 'View the Fleet handbook.';

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

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
    if (!thisPage) {
      throw 'notFound';
    }

    if (false) {
      // TODO: add "redirect" exit and handle mismatched capitalization / extra slashes by redirecting to the correct URL.  e.g. "http://localhost:2024/docs//usiNG-fleet///" Partial example of this: https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js#L161-L166
      let revisedUrl = 'todo';
      throw {redirect: revisedUrl};
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

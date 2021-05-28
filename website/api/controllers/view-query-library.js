module.exports = {


  friendlyName: 'View query library',


  description: 'Display "Query library" page.',


  exits: {
    success: { viewTemplatePath: 'pages/query-library' },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.queries)) {
      throw {badConfig: 'builtStaticContent.queries'};
    }

    // Respond with view.
    return {
      queries: sails.config.builtStaticContent.queries
    };

  }


};

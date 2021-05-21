module.exports = {


  friendlyName: 'View query library',


  description: 'Display "Query library" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/query-library'
    }

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.queries)) {
      throw new Error('Missing or invalid `sails.config.builtStaticContent.queries`!  Try doing `sails run build-static-content` and re-lift the server.');
    }

    // Respond with view.
    return {
      queries: sails.config.builtStaticContent.queries
    };

  }


};

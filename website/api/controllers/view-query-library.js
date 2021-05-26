module.exports = {


  friendlyName: 'View query library',


  description: 'Display "Query library" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/query-library'
    }

  },


  fn: async function () {

    if (!sails.config.builtStaticContent.queries) {
      throw new Error('Missing `sails.config.builtStaticContent.queries`!  Try doing `sails run build-static-content`.');
    }

    // Respond with view.
    return {
      queries: sails.config.builtStaticContent.queries
    };

  }


};

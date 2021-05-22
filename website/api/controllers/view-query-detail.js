module.exports = {


  friendlyName: 'View query detail',


  description: 'Display "Query detail" page.',


  inputs: {
    slug: { type: 'string', required: true, description: 'A slug uniquely identifying this query in the library.', example: 'get-macos-disk-free-space-percentage' },
  },


  exits: {
    success: { viewTemplatePath: 'pages/query-detail' },
    notFound: { responseType: 'notFound' },
  },


  fn: async function ({ slug }) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.queries)) {
      throw new Error('Missing or invalid `sails.config.builtStaticContent.queries`!  Try doing `sails run build-static-content` and re-lift the server.');
    }

    // Serve appropriate content for query.
    // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js
    let query = _.find(sails.config.builtStaticContent.queries, { slug: slug });
    if (!query) {
      throw 'notFound';
    }//•

    // Respond with view.
    return {
      query
    };

  }


};

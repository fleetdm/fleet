module.exports = {


  friendlyName: 'View benchmark detail',


  description: 'Display "Benchmark detail" page.',


  inputs: {
    slug: { type: 'string', required: true, description: 'A slug uniquely identifying this benchmark in the library.', example: 'get-macos-disk-free-space-percentage' },
  },


  exits: {
    success: { viewTemplatePath: 'pages/benchmark-detail' },
    notFound: { responseType: 'notFound' },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function ({ slug }) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.benchmarks)) {
      throw {badConfig: 'builtStaticContent.benchmarks'};
    } else if (!_.isString(sails.config.builtStaticContent.cisBenchmarkLibraryMacYmlRepoPath)) {
      throw {badConfig: 'builtStaticContent.cisBenchmarkLibraryMacLibraryYmlRepoPath'};
    }

    // Serve appropriate content for query.
    // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js
    let query = _.find(sails.config.builtStaticContent.queries, { slug: slug });
    if (!query) {
      throw 'notFound';
    }

    // Setting the meta title and description of this page using the query object, and falling back to a generic title or description if query.name or query.description are missing.
    let pageTitleForMeta = query.name ? query.name + ' | Query details' : 'Query details | Fleet for osquery';
    let pageDescriptionForMeta = query.description ? query.description : 'View more information about a query in Fleet\'s standard query library';
    // Respond with view.
    return {
      query,
      cisBenchmarkLibraryMacYmlRepoPath: sails.config.builtStaticContent.cisBenchmarkLibraryMacYmlRepoPath,
      pageTitleForMeta,
      pageDescriptionForMeta,
    };

  }


};

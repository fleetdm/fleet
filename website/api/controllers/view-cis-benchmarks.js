module.exports = {


  friendlyName: 'View CIS benchmarks',


  description: 'Display "CIS benchmarks" page.',


  exits: {
    success: { viewTemplatePath: 'pages/cis-benchmark-library' },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.cisBenchmarks)) {
      throw {badConfig: 'builtStaticContent.cisBenchmarks'};
    }

    // Respond with view.
    return {
      queries: sails.config.builtStaticContent.cisBenchmarks
    };

  }


};

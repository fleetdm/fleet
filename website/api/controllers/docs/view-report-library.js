module.exports = {


  friendlyName: 'View report library',


  description: 'Display "Report library" page.',


  exits: {
    success: { viewTemplatePath: 'pages/docs/report-library' },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.queries)) {
      throw {badConfig: 'builtStaticContent.queries'};
    }
    let policies = _.where(sails.config.builtStaticContent.queries, {kind: 'query'});
    let macOsQueries = _.filter(policies, (policy)=>{
      let platformsForThisPolicy = policy.platform.split(', ');
      return _.includes(platformsForThisPolicy, 'darwin');
    });
    let windowsQueries = _.filter(policies, (policy)=>{
      let platformsForThisPolicy = policy.platform.split(', ');
      return _.includes(platformsForThisPolicy, 'windows');
    });
    let linuxQueries = _.filter(policies, (policy)=>{
      let platformsForThisPolicy = policy.platform.split(', ');
      return _.includes(platformsForThisPolicy, 'linux');
    });
    // Respond with view.
    return {
      macOsQueries,
      windowsQueries,
      linuxQueries,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

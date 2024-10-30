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
    let policies = _.where(sails.config.builtStaticContent.queries, {kind: 'policy'});
    let macOsPolicies = _.filter(policies, (policy)=>{
      let platformsForThisPolicy = policy.platform.split(',');
      console.log(platformsForThisPolicy, _.includes(platformsForThisPolicy, 'darwin'));
      return _.includes(platformsForThisPolicy, 'darwin');
    });
    let windowsPolicies = _.filter(policies, (policy)=>{
      let platformsForThisPolicy = policy.platform.split(',');
      console.log(platformsForThisPolicy, _.includes(platformsForThisPolicy, 'darwin'));
      return _.includes(platformsForThisPolicy, 'windows');
    });
    let linuxPolicies = _.filter(policies, (policy)=>{
      let platformsForThisPolicy = policy.platform.split(',');
      console.log(platformsForThisPolicy, _.includes(platformsForThisPolicy, 'darwin'));
      return _.includes(platformsForThisPolicy, 'linux');
    });
    // console.log(macOsQueries);
    // Respond with view.
    return {
      macOsPolicies,
      windowsPolicies,
      linuxPolicies
    };

  }


};

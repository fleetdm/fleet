module.exports = {


  friendlyName: 'View policy library',


  description: 'Display "Policy library" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/policy-library'
    },
    badConfig: { responseType: 'badConfig' },

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.policies)) {
      throw {badConfig: 'builtStaticContent.policies'};
    }
    let policies = _.where(sails.config.builtStaticContent.policies, {kind: 'policy'});
    let macOsPolicies = _.filter(policies, (policy)=>{
      let platformsForThisPolicy = policy.platform.split(',');
      return _.includes(platformsForThisPolicy, 'darwin');
    });
    let windowsPolicies = _.filter(policies, (policy)=>{
      let platformsForThisPolicy = policy.platform.split(',');
      return _.includes(platformsForThisPolicy, 'windows');
    });
    let linuxPolicies = _.filter(policies, (policy)=>{
      let platformsForThisPolicy = policy.platform.split(',');
      return _.includes(platformsForThisPolicy, 'linux');
    });
    let chromePolicies = _.filter(policies, (policy)=>{
      let platformsForThisPolicy = policy.platform.split(',');
      return _.includes(platformsForThisPolicy, 'chrome');
    });
    // Respond with view.
    return {
      macOsPolicies,
      windowsPolicies,
      linuxPolicies,
      chromePolicies,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };


  }


};

module.exports = {


  friendlyName: 'View app library',


  description: 'Display "App library" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/app-library'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.appLibrary) || !sails.config.builtStaticContent.appLibrary) {
      throw {badConfig: 'builtStaticContent.appLibrary'};
    }

    let allApps = sails.config.builtStaticContent.appLibrary;
    allApps = _.sortBy(allApps, 'name');
    // Respond with view.
    return {allApps};

  }


};

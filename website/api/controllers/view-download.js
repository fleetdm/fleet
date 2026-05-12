module.exports = {


  friendlyName: 'View download',


  description: 'Display "Download" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/download'
    },
    badConfig: { responseType: 'badConfig' },

  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isObject(sails.config.builtStaticContent.fleetctlDownloadUrls)) {
      throw {badConfig: 'builtStaticContent.fleetctlDownloadUrls'};
    }

    let fleetctlDownloadUrls = sails.config.builtStaticContent.fleetctlDownloadUrls;


    return {
      macOsDownloadUrl: fleetctlDownloadUrls.macOs,
      windowsDownloadUrl: fleetctlDownloadUrls.windows,
      windowsArmDownloadUrl: fleetctlDownloadUrls.windowsArm,
    };

  }


};

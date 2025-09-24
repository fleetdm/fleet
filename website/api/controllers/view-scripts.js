module.exports = {


  friendlyName: 'View scripts',


  description: 'Display "Scripts" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/scripts'
    },
    badConfig: { responseType: 'badConfig' },

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.scripts)) {
      throw {badConfig: 'builtStaticContent.scripts'};
    }
    let scripts = sails.config.builtStaticContent.scripts;
    let macOsScripts = _.filter(scripts, (script)=>{
      return script.platform === 'macos';
    });
    let windowsScripts = _.filter(scripts, (script)=>{
      return script.platform === 'windows';
    });
    let linuxScripts = _.filter(scripts, (script)=>{
      return script.platform === 'linux';
    });
    // Respond with view.
    return {
      macOsScripts,
      windowsScripts,
      linuxScripts,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

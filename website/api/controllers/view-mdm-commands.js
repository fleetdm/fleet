module.exports = {


  friendlyName: 'View mdm commands',


  description: 'Display "Mdm commands" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/mdm-commands'
    },
    badConfig: { responseType: 'badConfig' },

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.mdmCommands)) {
      throw {badConfig: 'builtStaticContent.scripts'};
    }
    let commands = sails.config.builtStaticContent.mdmCommands;
    let appleCommands = _.filter(commands, (command)=>{
      return command.platform === 'apple';
    });
    let windowsCommands = _.filter(commands, (command)=>{
      return command.platform === 'windows';
    });

    let windowsCategories = _.groupBy(windowsCommands, 'category');
    let appleCategories = _.groupBy(appleCommands, 'category');
    // respond with view.
    return {
      windowsCategories,
      windowsCommands,
      appleCategories,
      appleCommands,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

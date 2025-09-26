module.exports = {


  friendlyName: 'View command details',


  description: 'Display "Command details" page.',


  inputs: {
    slug: { type: 'string', required: true, description: 'A slug uniquely identifying this script in the library.', example: 'macos-uninstall-fleetd' },
  },

  exits: {

    success: { viewTemplatePath: 'pages/command-details' },
    notFound: { responseType: 'notFound' },
    badConfig: { responseType: 'badConfig' },

  },


  fn: async function ({slug}) {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.mdmCommands)) {
      throw {badConfig: 'builtStaticContent.mdmCommands'};
    }
    let thisCommand = _.find(sails.config.builtStaticContent.mdmCommands, { slug: slug });
    if(!thisCommand) {
      throw 'notFound';
    }
    let pageTitleForMeta = ` ${thisCommand.name} command | Fleet controls library`;
    let pageDescriptionForMeta = `${thisCommand.description}`;
    // Respond with view.
    return {
      thisCommand,
      pageTitleForMeta,
      pageDescriptionForMeta,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

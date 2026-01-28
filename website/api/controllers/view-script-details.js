module.exports = {


  friendlyName: 'View script details',


  description: 'Display "Script details" page.',


  inputs: {
    slug: { type: 'string', required: true, description: 'A slug uniquely identifying this script in the library.', example: 'macos-uninstall-fleetd' },
  },


  exits: {

    success: { viewTemplatePath: 'pages/script-details' },
    notFound: { responseType: 'notFound' },
    badConfig: { responseType: 'badConfig' },

  },


  fn: async function ({slug}) {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.scripts)) {
      throw {badConfig: 'builtStaticContent.scripts'};
    }


    let thisScript = _.find(sails.config.builtStaticContent.scripts, { slug: slug });
    if(!thisScript){
      throw 'notFound';
    }

    let pageTitleForMeta = `${thisScript.name} | Fleet controls library`;
    let pageDescriptionForMeta = thisScript.description ? thisScript.description : 'View more information about a script in Fleet\'s controls library';

    // Respond with view.
    return {
      thisScript,
      pageTitleForMeta,
      pageDescriptionForMeta,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

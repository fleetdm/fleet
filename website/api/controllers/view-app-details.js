module.exports = {


  friendlyName: 'View app details',


  description: 'Display "App details" page.',


  inputs: {
    appIdentifier: {
      type: 'string',
      required: true,
      description: 'the identifier of an app in Fleet\'s maintained app library.',
      example: '1password'
    },
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/app-details'
    },

    badConfig: {
      responseType: 'badConfig'
    },

    notFound: {
      responseType: 'notFound'
    },

  },


  fn: async function ({appIdentifier}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.appLibrary) || !sails.config.builtStaticContent.appLibrary) {
      throw {badConfig: 'builtStaticContent.appLibrary'};
    }

    let thisApp = _.find(sails.config.builtStaticContent.appLibrary, { identifier: appIdentifier });
    if (!thisApp) {
      throw 'notFound';
    }
    // FUTURE: make these better.
    let pageTitleForMeta = thisApp.name + ' | Fleet app library';
    // let pageDescriptionForMeta = 'TODO'

    // Respond with view.
    return {
      thisApp,
      // pageDescriptionForMeta,
      pageTitleForMeta,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

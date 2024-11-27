module.exports = {


  friendlyName: 'View basic app',


  description: 'Display "Basic app" page.',


  inputs: {
    appIdentifier: {
      type: 'string',
      required: true,
      description: '',
      example: '1password'
    },
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/basic-app'
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
    // let pageDescriptionForMeta = ''

    // Respond with view.
    return {
      thisApp,
      // pageDescriptionForMeta,
      pageTitleForMeta,
    };

  }


};

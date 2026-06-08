module.exports = {


  friendlyName: 'View os settings',


  description: 'Display "Os settings" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/docs/os-settings'
    }

  },


  fn: async function () {

    // Respond with view.
    return {
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

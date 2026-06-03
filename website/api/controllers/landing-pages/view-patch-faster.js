module.exports = {


  friendlyName: 'View patch faster',


  description: 'Display "Patch faster" landing page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/patch-faster'
    },
    badConfig: { responseType: 'badConfig' },

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

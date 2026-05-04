module.exports = {


  friendlyName: 'View deployment',


  description: 'Display "Deployment" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/deployment'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

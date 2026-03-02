module.exports = {


  friendlyName: 'View linux management',


  description: 'Display "Linux management" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/linux-management'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

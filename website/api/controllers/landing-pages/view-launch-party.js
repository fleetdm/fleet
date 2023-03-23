module.exports = {


  friendlyName: 'View launch party',


  description: 'Display "Launch party" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/launch-party'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'View landing',


  description: 'Display "Landing" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

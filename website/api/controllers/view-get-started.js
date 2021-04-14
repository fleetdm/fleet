module.exports = {


  friendlyName: 'View get started',


  description: 'Display "Get started" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/get-started'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

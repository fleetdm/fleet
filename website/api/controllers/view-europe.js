module.exports = {


  friendlyName: 'View europe',


  description: 'Display "Europe" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/europe'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

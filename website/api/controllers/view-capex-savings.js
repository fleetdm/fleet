module.exports = {


  friendlyName: 'View capex savings',


  description: 'Display "Capex savings" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/capex-savings'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

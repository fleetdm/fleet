module.exports = {


  friendlyName: 'View pricing',


  description: 'Display "Pricing" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/pricing'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'View deals',


  description: 'Display "Deals" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/deals'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

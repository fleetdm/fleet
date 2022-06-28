module.exports = {


  friendlyName: 'View sandbox',


  description: 'Display "Sandbox" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/sandbox'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'View aviation',


  description: 'Display "Aviation" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/aviation'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

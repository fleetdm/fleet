module.exports = {


  friendlyName: 'View unused software',


  description: 'Display "Unused software" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/imagine/unused-software'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

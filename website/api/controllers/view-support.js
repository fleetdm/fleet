module.exports = {


  friendlyName: 'View support',


  description: 'Display "Support" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/support'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

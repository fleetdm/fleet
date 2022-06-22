module.exports = {


  friendlyName: 'View forgot password',


  description: 'Display "Forgot password" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/forgot-password'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'View sandbox login',


  description: 'Display "Sandbox login" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/sandbox-login'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

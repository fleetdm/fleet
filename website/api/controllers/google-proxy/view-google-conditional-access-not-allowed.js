module.exports = {


  friendlyName: 'View google conditional access not allowed',


  description: 'Display "Google conditional access not allowed" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/google-proxy/google-conditional-access-not-allowed'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

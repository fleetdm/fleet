module.exports = {


  friendlyName: 'View google conditional access not found',


  description: 'Display "Google conditional access not found" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/google-proxy/google-conditional-access-not-found'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

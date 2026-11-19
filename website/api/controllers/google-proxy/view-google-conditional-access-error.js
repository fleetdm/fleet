module.exports = {


  friendlyName: 'View google conditional access error',


  description: 'Display "Google conditional access error" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/google-proxy/google-conditional-access-error'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

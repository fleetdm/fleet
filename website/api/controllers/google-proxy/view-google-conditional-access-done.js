module.exports = {


  friendlyName: 'View google conditional access done',


  description: 'Display "Google conditional access done" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/google-proxy/google-conditional-access-done'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

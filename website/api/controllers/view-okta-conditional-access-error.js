module.exports = {


  friendlyName: 'View okta conditional access error',


  description: 'Display "Okta conditional access error" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/okta-conditional-access-error'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

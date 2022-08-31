module.exports = {


  friendlyName: 'View Sandbox login',


  description: 'Display the "Sandbox Login" page. Note: This page is the "login" page skinned for Fleet Sandbox.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/sandbox-login'
    },

    redirect: {
      description: 'The requesting user is already logged in.',
      responseType: 'redirect'
    }


  },


  fn: async function () {

    // If the user is logged in, redirect them to the Fleet sandbox page.
    if (this.req.me) {
      throw {redirect: '/try-fleet/sandbox'};
    }

    // Respond with view.
    return {};

  }


};

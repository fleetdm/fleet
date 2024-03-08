module.exports = {


  friendlyName: 'View register',


  description: 'Display "Register" page. Note: This page is the "signup" page skinned for Fleet Sandbox.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/register'
    },

    redirect: {
      description: 'The requesting user is already logged in.',
      responseType: 'redirect'
    }

  },


  fn: async function () {

    // If the user is logged in, redirect them to the Fleet sandbox page.
    // FUTURE: once all Sandbox instances have expired, redirect users to the fleetctl-preview page.
    if (this.req.me) {
      throw {redirect: '/try-fleet/sandbox'};
    }

    // Respond with view.
    return {};

  }


};

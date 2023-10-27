module.exports = {


  friendlyName: 'View fleetctl preview',


  description: 'Display "fleetctl preview" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/fleetctl-preview'
    },

    redirect: {
      description: 'The requesting user is not logged in.',
      responseType: 'redirect'
    },

  },


  fn: async function () {

    // Note: This page bypasses the 'is-logged-in' policy so we can redirect not-logged-in users to the /try-fleet/login page,
    if(!this.req.me){
      throw {redirect: '/try-fleet/login' };
    }
    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'View new sandbox',


  description: 'Display "New sandbox" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/new-sandbox'
    },

    redirect: {
      description: 'The logged in user has already provisioned a Fleet Sandbox instance',
      responseType: 'redirect'
    }

  },


  fn: async function () {

    if(this.req.me.fleetSandboxURL) {
      throw {redirect: '/try-fleet/sandbox'};
    }
    // Respond with view.
    return;

  }


};

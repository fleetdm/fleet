module.exports = {


  friendlyName: 'View sandbox',


  description: 'Display "Sandbox" page or redirect users to their Fleet Sandbox instance.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/log-into-fleet-sandbox-and-redirect',
      description: 'This user is being logged into their Fleet Sandbox instance.'
    },

    redirect: {
      description: 'This user\'s Fleet Sandbox instance is expired.',
      responseType: 'redirect'
    },

  },


  fn: async function () {

    if(!this.req.me.fleetSandboxURL) {
      throw new Error(`Consistency violation: The logged-in user's (${this.req.me.emailAddress}) fleetSandboxURL has somehow gone missing!`);
    }

    if(!this.req.me.fleetSandboxExpiresAt) {
      throw new Error(`Consistency violation: The logged-in user's (${this.req.me.emailAddress}) fleetSandboxExpiresAt has somehow gone missing!`);
    }

    if(!this.req.me.fleetSandboxDemoKey) {
      throw new Error(`Consistency violation: The logged-in user's (${this.req.me.emailAddress}) fleetSandboxDemoKey has somehow gone missing!`);
    }

    // If this user's Fleet Sandbox instance is expired, we'll redirect them to the sandbox-expired page
    if(this.req.me.fleetSandboxExpiresAt < Date.now()){
      throw {redirect: '/try-fleet/sandbox-expired' };
    }

    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'View sandbox',


  description: 'Display "Sandbox" page or redirect users to their Fleet Sandbox instance.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/log-into-fleet-sandbox-and-redirect',
      description: 'This user is being logged into their Fleet Sandbox instance.'
    },

    redirect: {
      description: 'This user does not have a Fleet Sandbox instance, or their instance has expired.',
      responseType: 'redirect'
    }


  },


  fn: async function () {

    // If the user is not logged in, redirect them to the Fleet sandbox registration page.
    if (!this.req.me) {
      throw {redirect: '/try-fleet/register'};
    }


    // Check if the user has a fleetSandboxURL
    if(!this.req.me.fleetSandboxURL) {
      // If the user doesn't have a fleetSandboxURL they will be taken to the sandbox page.
      throw {redirect: '/try-fleet/new-sandbox'};
    } else {
      // If this user's Fleet Sandbox instance is expired, we'll show the sandbox page with sandboxExpired: true
      if(this.req.me.fleetSandboxExpiresAt < Date.now()) {
        throw {redirect: '/try-fleet/sandbox-expired' };
      }
      // Get the userRecord so we can send their hashed password to the sandbox instance
      let sandboxUser = await User.findOne({id: this.req.me.id});

      let sandboxURL = this.req.me.fleetSandboxURL;

      // If this is a valid fleet sandbox instance, we'll check the /healthz endpoint before redirecting the user to their sandbox.
      await sails.helpers.flow.until(async()=>{
        let serverResponse = await sails.helpers.http.sendHttpRequest('GET', sandboxURL+'/healthz')
        .timeout(5000)
        .tolerate('non200Response')
        .tolerate('requestFailed');
        if(serverResponse) {
          return serverResponse.statusCode === 200;
        }
      });
      // Respond with view.
      return {
        sandboxUser,
      };
    }

  }


};

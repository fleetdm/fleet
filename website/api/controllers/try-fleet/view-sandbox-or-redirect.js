module.exports = {


  friendlyName: 'View sandbox',


  description: 'Display "Sandbox" page or redirect users to their Fleet Sandbox instance.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/sandbox'
    },

    redirect: {
      description: 'The user has been redirected to their Fleet sandbox instance',
      responseType: 'redirect'
    }

  },


  fn: async function () {

    // If the user is not logged in, redirect them to the Fleet sandbox registration page.
    if (!this.req.me) {
      throw {redirect: '/try-fleet/register'};
    }


    // Check if the user has a fleetSandboxURL
    if(this.req.me.fleetSandboxURL) {

      // Check if this sandbox instance is expired.
      if(this.req.me.fleetSandboxExpiresAt > Date.now()) {
        // Setting this.req.me.fleetSandboxURL to a variable to pass in to sails.helper.flow.until()
        let sandboxURL = this.req.me.fleetSandboxURL;
        // If this is a valid fleet sandbox instance, we'll check the /healthz endpoint before redirecting the user to their sandbox.
        await sails.helpers.flow.until(async function () {
          let serverResponse = await sails.helpers.http.sendHttpRequest('GET', sandboxURL+'/healthz').timeout(5000).tolerate('non200Response').tolerate('requestFailed');
          if(serverResponse) {
            return serverResponse.statusCode === 200;
          }
        });
        throw {redirect: this.req.me.fleetSandboxURL+'?demoKey='+this.req.me.fleetSandboxDemoKey};
      }
    }


    // If the user doesn't have a fleetSandboxURL, or their Fleet Sandbox instance is expired, they will be taken to the sandbox page.
    // Respond with view.
    return;
  }


};

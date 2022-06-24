module.exports = {


  friendlyName: 'View sandbox',


  description: 'Display "Sandbox" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/sandbox'
    },

    redirect: {
      description: 'The requesting user is already logged in.',
      responseType: 'redirect'
    }

  },


  fn: async function () {

    // If the user is not logged in, redirect them to the Fleet sandbox registration page.
    // if (!this.req.me) {
    //   throw {redirect: '/try-fleet/register'};
    // }

    // Check if the user has a fleetSandboxURL (this.req.me.fleetSandboxUrl)
      // If the user doesn't have a fleetSandboxURL, they will be taken to the sandbox page with an empty fleetSandboxURL and the isFleetSandboxExpired flag set to false.

    // Check the fleetSandboxExpiresAt (this.req.me.fleetSandboxExpiresAt)
      // If the sandbox instance is expired, we'll set a flag to display the sandbox expired state of the sandbox page (isFleetSandboxExpired: true)
      // If the sandbox instance has not expired, we'll check the /healthz endpoint.
        // Note: we're only checking this enpoint once here, all other checks will be handled by /try-fleet/redirect-to-fleet-sandbox
        // If the /healthz endpoint returns a 200 response, we'll redirect the user to their Fleet sandbox instance.

        // If the sandbox instance is not ready yet, we'll take the user the sandbox page and return the fleetSandboxURL.

    // Respond with view.
    return {
      // isFleetSandboxExpired,
      // fleetSandboxURL
    };

  }


};

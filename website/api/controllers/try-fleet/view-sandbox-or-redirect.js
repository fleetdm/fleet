module.exports = {


  friendlyName: 'View sandbox',


  description: 'Display "Sandbox" page or redirect users to their Fleet Sandbox instance.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/sandbox'
    },

    redirect: {
      description: 'This user does not have a Fleet Sandbox instance.',
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
      throw {redirect: '/try-fleet/new-sandbox'}
    } else {
      // Get the userRecord to send to the
      let userRecord = await User.findOne({id: this.req.me.id});

      // Setting this.req.me.fleetSandboxURL to a variable to pass in to sails.helper.flow.until()
      let sandboxURL = this.req.me.fleetSandboxURL;
      // If this is a valid fleet sandbox instance, we'll check the /healthz endpoint before redirecting the user to their sandbox.
      await sails.helpers.flow.until(async function () {
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
        sandboxUser: userRecord,
      };
    }

  }


};

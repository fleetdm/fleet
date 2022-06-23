module.exports = {


  friendlyName: 'View sandbox',


  description: 'Display "Sandbox" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/sandbox'
    }

  },


  fn: async function () {

    // If the user is not logged in, redirect them to the Fleet sandbox registration page.
    // if (!this.req.me) {
    //   throw {redirect: '/try-fleet/register'};
    // }

    // Make sure this user has a fleetSandboxURL
    // if(!this.req.me.fleetSandboxURL) {
        // If they don't have a fleetSandboxURL, we'll redirect this user to the change password page
        // If their password has been updated to meet the new requirements, we'll provision them a Fleet sandbox instance
        // Create an ISO timestamp set 24 hours from now
        // call the provision-fleet-sandbox helper, passing in the created timestamp and the logged in user's ID
    // }

    //

    // Respond with view.
    return {};

  }


};

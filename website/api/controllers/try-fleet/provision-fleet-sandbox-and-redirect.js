module.exports = {


  friendlyName: 'Provision fleet sandbox and redirect',


  description: '',


  inputs: {

    // User ID

  },


  exits: {

    // Error: User already has valid fleet sandbox

    // Success: A user's fleet sandbox instance has been provisioned and is ready to be used.

  },


  fn: async function (inputs) {

    // Find user record (User.findOne({id: userID}))
      // If the user has a fleetSandboxURL, throw an error
    // Call the provision sandbox helper and get the URL of the Fleet sandbox instance (fleetSandboxInfo = await sails.helpers.provisionFleetSandbox.with(userId: user.id));

    // Once we have the URL, we'll check the /healthz endpoint and return the Fleet sandbox url when it returns a 200 status (await sails.helpers.flow.until(......))


    // When the Fleet Sandbox instance is ready, return the sandbox url
    // return sandboxURL;

  }


};

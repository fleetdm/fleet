module.exports = {


  friendlyName: 'Provision fleet sandbox',


  description: '',


  inputs: {

    // userID: {}

    // sandboxExpirationTimestamp: { }

  },


  exits: {

    success: {
      description: 'All done.',
    },

  },


  fn: async function (inputs) {

    // Find user record using User ID
    // let user = User.find(....)

    // Check for a fleetSandboxURL on the user record
    // If(user.fleetSandboxURL) {
    // If the user has a fleetSandboxURL, throw an error

    // Send POST request to the cloud provisioner API
    // let cloudProvisionerResponse = await sails.helpers.http.post('CLOUD_PROVISIONER_API_URL', {REQUEST_DATA}).timeout(5000).interpect(TODO)
    //

    // Attach the URL and demoKey in the response to the user record

    // If the request was successful, update the user record with the fleetSandboxURL and sandboxExpirationTimestamp
    // await User.updateOne({id: user.id}).set({
    //   fleetSandboxURL: cloudProvisionerResponse.url,
    //   fleetSandboxExpiresAt: Date.parse(sandboxExpirationTimestamp), // Note: We store the expiration timestamp as a JS timestamp, instead of an ISO string.
    // });
    // return the fleetSandboxURL
    // return fleetSandboxURL;
  }


};


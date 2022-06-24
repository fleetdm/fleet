module.exports = {


  friendlyName: 'Provision fleet sandbox',


  description: '',


  inputs: {

    // userID: {}

    // sandboxExpiresAt: {}

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

    // If a sandboxExpiresAt was provided, we'll use that value, otherwise we'll create a new timestamp. Why?
    // let expiresAt;
    // if(!inputs.sandboxExpiresAt) {
    //   expiresAt = New Date(Date.now() + (24*60*60*1000)).toISOString();
    // } else {
    //   expiresAt = inputs.sandboxExpiresAt;
    // }
    // Note: Both this helper and The cloud provisioner API expect an ISO 8601 string timestamp, but we store the timestamp in the website database as a JS timestamp

    // Send POST request to the cloud provisioner API
    // let cloudProvisionerResponse = await sails.helpers.http.post('CLOUD_PROVISIONER_API_URL', {
      // API Request data
    // }).timeout(5000).intercept(TODO);


    // If the request was successful, update the user record with the fleetSandboxURL and sandboxExpiresAt
    // await User.updateOne({id: user.id}).set({
    //   fleetSandboxURL: cloudProvisionerResponse.url,
    //   fleetSandboxExpiresAt: Date.parse(sandboxExpiresAt), // Note: We store the expiration timestamp as a JS timestamp, instead of an ISO string.
    // });

    // return fleetSandboxURL;
  }


};


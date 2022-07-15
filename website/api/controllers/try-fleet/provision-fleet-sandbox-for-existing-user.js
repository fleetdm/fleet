module.exports = {


  friendlyName: 'Provision Fleet Sandbox for existing user',


  description: 'This action requests a new Fleet Sandbox instance and polls the `/healthz` endpoint until the new instance is available',


  inputs: {

    userID: {
      type: 'string',
      required: true,
    }


  },


  exits: {

    success: {
      decription: 'The user has successfully provisioned a Fleet Sandbox Instanace and has been redirected to it'
    },

    userHasExistingSandbox: {
      decription: 'This user has already provisioned a Fleet Sandbox Instance'
    },

    couldNotProvisionSandbox: {
      description: 'An error occurred while trying to provision the Fleet Sandbox Instance'
    }
  },


  fn: async function ({userID}) {

    // Find user record
    let user = await User.findOne({id: userID});
    if(user.fleetSandboxURL) {
      return 'userHasExistingSandbox';
    }

    // Creating an expiration JS timestamp for the Fleet sandbox instance. NOTE: We send this value to the cloud provisioner API as an ISO 8601 string.
    let fleetSandboxExpiresAt = Date.now() + (24*60*60*1000);

    // Send a POST request to the cloud provisioner API
    let cloudProvisionerResponse = await sails.helpers.http.post('https://sandbox.fleetdm.com/new', {
      'name': user.firstName + ' ' + user.lastName,
      'email': user.emailAddress,
      'password': user.password,
      'sandbox_expiration': new Date(fleetSandboxExpiresAt).toISOString(), // sending expiration_timestamp as an ISO string.
    })
    .timeout(5000)
    .intercept('non200Response', 'couldNotProvisionSandbox');

    if(!cloudProvisionerResponse.URL) {
      throw 'couldNotProvisionSandbox';
    } else {
      // Update this user's record with the fleetSandboxURL and fleetSandboxExpiresAt
      await User.updateOne({id: user.id}).set({
        fleetSandboxURL: cloudProvisionerResponse.URL,
        fleetSandboxExpiresAt: fleetSandboxExpiresAt,
      });
      // Poll the Fleet Sandbox Instance's /healthz endpoint until it returns a 200 response
      await sails.helpers.flow.until( async()=>{
        let serverResponse = await sails.helpers.http.sendHttpRequest('GET', cloudProvisionerResponse.URL+'/healthz').timeout(5000).tolerate('non200Response').tolerate('requestFailed');
        if(serverResponse && serverResponse.statusCode) {
          return serverResponse.statusCode === 200;
        }
      });
    }

    return;
  }


};

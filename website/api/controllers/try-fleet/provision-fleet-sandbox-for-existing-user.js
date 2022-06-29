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

    // Find user record (User.findOne({id: userID}))
    let user = await User.findOne({id: userID}).decrypt();
    if(user.fleetSandboxURL) {
      return 'userHasExistingSandbox';
    }

    // Creating an expiration JS timestamp for the Fleet sandbox instance. NOTE: We send this value to the cloud provisioner API as an ISO 8601 string.
    let fleetSandboxExpiresAt = Date.now() + (24*60*60*1000);

    // Create a key to send to the Fleet Sandbox instance, This key will be provided when the user logs in to their fleet sandbox instance
    let fleetSandboxDemoKey = await sails.helpers.strings.random('url-friendly');

    // Send a POSt request to the cloud provisioner API
    let cloudProvisionerResponse = await sails.helpers.http.post(sails.config.custom.fleetSandboxProvisionerURL, {
      'name': user.firstName + ' ' + user.lastName,
      'email': user.emailAddress,
      'password': user.password,
      'sandbox_expiration': new Date(user.fleetSandboxExpiresAt).toISOString(), // sending expiration_timestamp as an ISO string.
      'fleetSandboxKey': user.fleetSandboxKey,
      'apiSecret': sails.config.custom.fleetSandboxProvisionerSecret,
    })
    .timeout(5000)
    .intercept('non200Response', 'couldNotProvisionSandbox');

    let fleetSandboxURL;
    if(cloudProvisionerResponse.url) {
      fleetSandboxURL = cloudProvisionerResponse.url;

      // Update this user's record with the fleetSandboxURL, fleetSandboxExpiresAt, and fleetSandboxKey.
      await User.updateOne({id: user.id}).set({
        fleetSandboxURL: cloudProvisionerResponse.url,
        fleetSandboxExpiresAt: fleetSandboxExpiresAt,
        fleetSandboxDemoKey: fleetSandboxDemoKey,
      });

    // Poll the Fleet Sandbox Instance's /healthz endpoint until it returns a 200 response
      await sails.helpers.flow.until(async function () {
        let serverResponse = await sails.helpers.http.sendHttpRequest('GET', cloudProvisionerResponse.url+'/healthz').timeout(5000).tolerate('non200Response').tolerate('requestFailed');
        if(serverResponse) {
          return serverResponse.statusCode === 200;
        }
      });
    } else {
      throw 'couldNotProvisionSandbox';
    }

    // When the Fleet Sandbox instance is ready, we'll return the fleetSandboxUrl and the fleetSandboxKey
    return {
      fleetSandboxURL,
      fleetSandboxDemoKey
    }

  }


};

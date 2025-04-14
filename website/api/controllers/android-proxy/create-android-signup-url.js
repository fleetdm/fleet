module.exports = {


  friendlyName: 'Create android signup url',


  description: 'Creates and returns a signup URL for an android enterprise.',


  inputs: {
    projectId: {
      type: 'string',
      required: true,
    },
    callbackUrl: {
      type: 'string',
      required: true,
    }
  },


  exits: {
    success: { description: 'A signup URL has been sent to the requesting Fleet server.'}
  },


  fn: async function ({fleetServerSecret, androidEnterpriseId, projectId, callbackUrl}) {


    // Parse the fleet server url from the origin header.
    let fleetServerUrl = this.req.get('Origin');
    if(!fleetServerUrl){
      return this.res.badRequest();
    }

    // Check the databse for a record of this enterprise.
    let connectionforThisInstanceExists = await AndroidEnterprise.findOne({fleetServerUrl: fleetServerUrl});

    if(connectionforThisInstanceExists){
      throw 'enterpriseAlreadyExists';
    }

    let newFleetServerSecret = await sails.helpers.strings.random.with({len: 30});

    let authorizationTokenForThisRequest = await sails.helpers.androidEnterprise.getAccessToken.with({
      // TODO: this helper doesn't exist
    });


    // Send a request to delete the Android enterprise.
    let createSignupUrlResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `https://androidmanagement.googleapis.com/v1/signupUrls?projectId=${projectId}&callbackUrl=${callbackUrl}`,
      headers: {
        Authorization: `Bearer ${authorizationTokenForThisRequest}`,
      },
    });

    // Create a databse record for the newly created enterprise
    await AndroidEnterprise.createOne({
      fleetServerUrl: fleetServerUrl,
      fleetServerSecret: newFleetServerSecret,
      // fleetLicenseKey: fleetLicenseKey,
      // androidEnterpriseId: newAndroidEnterpriseId
    });



    return {
      signup_url: createSignupUrlResponse.url,
      signup_url_name: createSignupUrlResponse.name,
      fleet_server_secret: newFleetServerSecret
    };



  }


};

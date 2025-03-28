module.exports = {


  friendlyName: 'Create android signup url',


  description: 'Creates and returns a signup URL for an android enterprise.',


  inputs: {
    fleetServerSecret: {
      type: 'string',
      required: true,
    },
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
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

    // Authenticate this request
    let thisAndroidEnterprise = await AndroidEnterprise.findOne({
      fleetServerSecret: fleetServerSecret,
      androidEnterpriseId: androidEnterpriseId,
    });

    // Return a 404 response if no records are found.
    if(!thisAndroidEnterprise) {
      return this.res.notFound();
    }

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


    return createSignupUrlResponse;

  }


};

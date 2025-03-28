module.exports = {


  friendlyName: 'Modify android policies',


  description: '',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    profileId: {
      type: 'string',
      required: true,
    },
    fleetServerSecret: {
      type: 'string',
    },
    policy: {
      type: {},
      moreInfoUrl: 'https://developers.google.com/android/management/reference/rest/v1/enterprises.policies#Policy'
    }
  },


  exits: {

  },


  fn: async function ({androidEnterpriseId, profileId, fleetServerSecret, policy}) {

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
    let patchProfileResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'PATCH',
      url: `https://androidmanagement.googleapis.com/v1/enterprises/${androidEnterpriseId}/policies/${profileId}.`,
      body: policy,
      headers: {
        Authorization: `Bearer ${authorizationTokenForThisRequest}`,
      },
    });


    // All done.
    return patchProfileResponse;// ?

  }


};

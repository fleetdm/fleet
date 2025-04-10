module.exports = {


  friendlyName: 'Create android enrollment token',


  description: '',


  inputs: {
    fleetServerSecret: {
      type: 'string',
      required: true,
    },
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    enrollmentToken: {
      type: {},
      required: true,
      moreInfoUrl: 'https://developers.google.com/android/management/reference/rest/v1/enterprises.enrollmentTokens#resource:-enrollmenttoken'
    },
  },


  exits: {

  },


  fn: async function ({fleetServerSecret, androidEnterpriseId, enrollmentToken}) {
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
    let createEnrollmentTokenResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `https://androidmanagement.googleapis.com/v1/enterprises/${androidEnterpriseId}/enrollmentTokens`,
      body: enrollmentToken,
      headers: {
        Authorization: `Bearer ${authorizationTokenForThisRequest}`,
      },
    });



    // All done?
    return createEnrollmentTokenResponse.value;// ?

  }


};

module.exports = {


  friendlyName: 'Delete one android enterprise',


  description: 'Deletes an android enterprise and the associated database record.',


  inputs: {
    fleetServerSecret: {
      type: 'string',
      required: true,
    },
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'An Android enterprise was successfully deleted.' }
  },


  fn: async function ({fleetServerSecret, androidEnterpriseId}) {

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
    let deleteEnterpriseResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'DELETE',
      url: `https://androidmanagement.googleapis.com/v1/enterprises/${androidEnterpriseId}`,
      headers: {
        Authorization: `Bearer ${authorizationTokenForThisRequest}`,
      },
    });

    await AndroidEnterprise.destroyOne({ id: thisAndroidEnterprise.id });


    // All done.
    return;

  }


};

module.exports = {


  friendlyName: 'Modify android policies',


  description: 'Modifies a policy of an Android enterprise',


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
      description: 'The policy on the Android enterprise that is being updated.',
      moreInfoUrl: 'https://developers.google.com/android/management/reference/rest/v1/enterprises.policies#Policy'
    },
  },


  exits: {
    success: { description: 'The policy of an Android enterprise was successfully updated.' }
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
    // Update the policy for this Android enterprise.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occurr using google's node library.
    let modifyPoliciesResponse = await sails.helpers.flow.build(async ()=>{
      let google = require('googleapis');
      let androidmanagement = google.androidmanagement('v1');
      let googleAuth = new google.auth.GoogleAuth({
        scopes: ['https://www.googleapis.com/auth/androidmanagement'],
        credentials: {
          client_email: sails.config.custom.GoogleClientId,// eslint-disable-line camelcase
          private_key: sails.config.custom.GooglePrivateKey,// eslint-disable-line camelcase
        },
      });
      // Acquire the google auth client, and bind it to all future calls
      let authClient = await googleAuth.getClient();
      google.options({auth: authClient});
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises$Policies.html#patch
      let patchPoliciesResponse = await androidmanagement.enterprises.policies.patch({
        name: `enterprises/${androidEnterpriseId}/policies/${profileId}`,
        requestBody: policy
      });
      return patchPoliciesResponse.data;
    }).intercept((err)=>{
      return new Error(`When attempting to update a policy for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    });


    // Return the modified policy back to the Fleet server.
    return modifyPoliciesResponse;

  }


};

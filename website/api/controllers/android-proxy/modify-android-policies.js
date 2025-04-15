module.exports = {


  friendlyName: 'Modify android policies',


  description: 'Modifies a policy of an Android enterprise',


  inputs: {
    androidEnterpriseId: {
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
    }
  },


  exits: {
    success: { description: 'The policy of an Android enterprise was successfully updated.' }
  },


  fn: async function ({androidEnterpriseId, fleetServerSecret, policy}) {

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
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises$Enrollmenttokens.html#create
      let patchPoliciesResponse = await androidmanagement.enterprises.policies.patch({
        name: policy.name,// TODO: make sure this exists.
        // name: `enterprises/${androidEnterpriseId}/policies/default`,
        // updateMask: 'placeholder-value',// TODO: do we need to set an updateMask value? otherwise, the entire policy will be replaced with the provided policy.
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

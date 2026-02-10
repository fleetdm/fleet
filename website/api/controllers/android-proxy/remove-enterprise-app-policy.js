module.exports = {


  friendlyName: 'Remove android enterprise policy applications',


  description: 'Removes applications in an Android enterprise policy',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    policyId: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'The policy applications of an Android enterprise was successfully updated.' },
    missingAuthHeader: { description: 'This request was missing an authorization header.', responseType: 'unauthorized'},
    unauthorized: { description: 'Invalid authentication token.', responseType: 'unauthorized'},
    notFound: { description: 'No Android enterprise found for this Fleet server.', responseType: 'notFound'},
  },


  fn: async function ({ androidEnterpriseId, policyId}) {

    // Extract fleetServerSecret from the Authorization header
    let authHeader = this.req.get('authorization');
    let fleetServerSecret;

    if (authHeader && authHeader.startsWith('Bearer')) {
      fleetServerSecret = authHeader.replace('Bearer', '').trim();
    } else {
      throw 'missingAuthHeader';
    }

    // Authenticate this request
    let thisAndroidEnterprise = await AndroidEnterprise.findOne({
      androidEnterpriseId: androidEnterpriseId
    });

    // Return a 404 response if no records are found.
    if (!thisAndroidEnterprise) {
      throw 'notFound';
    }
    // Return an unauthorized response if the provided secret does not match.
    if (thisAndroidEnterprise.fleetServerSecret !== fleetServerSecret) {
      throw 'unauthorized';
    }

    // Check the list of Android Enterprises managed by Fleet to see if this Android Enterprise is still managed.
    let isEnterpriseManagedByFleet = await sails.helpers.androidProxy.getIsEnterpriseManagedByFleet(androidEnterpriseId);
    // Return a 404 response if this Android enterprise is no longer managed by Fleet.
    if(!isEnterpriseManagedByFleet) {
      throw 'notFound';
    }

    // Remove the policy applications for this Android enterprise.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occurr using google's node library.
    let removeApplicationPolicyResponse = await sails.helpers.flow.build(async () => {
      let { google } = require('googleapis');
      let androidmanagement = google.androidmanagement('v1');
      let googleAuth = new google.auth.GoogleAuth({
        scopes: ['https://www.googleapis.com/auth/androidmanagement'],
        credentials: {
          client_email: sails.config.custom.androidEnterpriseServiceAccountEmailAddress,// eslint-disable-line camelcase
          private_key: sails.config.custom.androidEnterpriseServiceAccountPrivateKey,// eslint-disable-line camelcase
        },
      });
      // Acquire the google auth client, and bind it to all future calls
      let authClient = await googleAuth.getClient();
      google.options({ auth: authClient });

      let response = await androidmanagement.enterprises.policies.removePolicyApplications({
        name: `enterprises/${androidEnterpriseId}/policies/${policyId}`,
        requestBody: this.req.body,
      });
      return response.data;
    }).intercept((err) => {
      return new Error(`When attempting to remove applications for a policy of Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    });


    // Return the modified policy back to the Fleet server.
    return removeApplicationPolicyResponse;

  }


};

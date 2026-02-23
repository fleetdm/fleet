module.exports = {


  friendlyName: 'Modify android policies',


  description: 'Modifies a policy of an Android enterprise',


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
    success: { description: 'The policy of an Android enterprise was successfully updated.' },
    missingAuthHeader: { description: 'This request was missing an authorization header.', responseType: 'unauthorized'},
    unauthorized: { description: 'Invalid authentication token.', responseType: 'unauthorized'},
    notFound: { description: 'No Android enterprise found for this Fleet server.', responseType: 'notFound'},
    invalidPolicy: { description: 'Invalid patch policy request', responseType: 'badRequest' },
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

    // Update the policy for this Android enterprise.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occurr using google's node library.
    let modifyPoliciesResponse = await sails.helpers.flow.build(async () => {
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
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises$Policies.html#patch
      let patchPoliciesResponse = await androidmanagement.enterprises.policies.patch({
        name: `enterprises/${androidEnterpriseId}/policies/${policyId}`,
        requestBody: this.req.body,
        updateMask: this.req.param('updateMask') // Pass the update mask to avoid overwriting applications
      });
      return patchPoliciesResponse.data;
    }).intercept({ status: 429 }, (err) => {
      // If the Android management API returns a 429 response, log an additional warning that will trigger a help-p1 alert.
      sails.log.warn(`p1: Android management API rate limit exceeded!`);
      return new Error(`When attempting to update a policy for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    }).intercept({ status: 400 }, (err) => {
      return {'invalidPolicy': `Attempted to update a policy with an invalid value for an Android enterprise (${androidEnterpriseId}): ${err}`};
    }).intercept((err) => {
      return new Error(`When attempting to update a policy for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    });


    // Return the modified policy back to the Fleet server.
    return modifyPoliciesResponse;

  }


};

module.exports = {


  friendlyName: 'Modify android enterprise policy applications',


  description: 'Modifies applications in an Android enterprise policy',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    policyId: {
      type: 'string',
      required: true,
    },
    googleAction: {
      type: 'string',
      defaultsTo: 'modifyPolicyApplications',
    },
    // packageNames is the body for the removePolicyApplications googleAction.
    packageNames: {
      type: ['string'],
    },
    // changes is the body for the modifyPolicyApplications googleAction.
    changes: {
      type: [{}],
    },
  },


  exits: {
    success: { description: 'The policy applications of an Android enterprise was successfully updated.' },
    missingAuthHeader: { description: 'This request was missing an authorization header.', responseType: 'unauthorized'},
    unauthorized: { description: 'Invalid authentication token.', responseType: 'unauthorized'},
    notFound: { description: 'No Android enterprise found for this Fleet server.', responseType: 'notFound'},
    enterpriseNotAccessible: { description: 'Fleet is not authorized to manage this Android enterprise.', responseType: 'notFound' },
    policyNotFound: { description: 'Specified policy not found', responseType: 'notFound' },
    managementApiError: { statusCode: 503, description: 'The Android management API returned a transient 5xx error.' },
  },


  fn: async function ({ androidEnterpriseId, policyId, googleAction, packageNames, changes }) {

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


    // Get the shared Google API auth client with the getAndroidManagementAuthorizationClient helper.
    // Note: we are doing this outside of the sails.helpers.flow.build() so any errors related to the website's credentials returned by the helper are not intercepted.
    let androidManagementAuthClient = await sails.helpers.androidProxy.getAndroidManagementAuthorizationClient();

    // Update the policy applications for this Android enterprise.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occurr using google's node library.
    let modifyApplicationPolicyResponse = await sails.helpers.flow.build(async () => {
      let { google } = require('googleapis');
      let androidManagementConnection = google.androidmanagement({version: 'v1', auth: androidManagementAuthClient});

      switch (googleAction) {
        case 'removePolicyApplications': {
          let response = await androidManagementConnection.enterprises.policies.removePolicyApplications({
            name: `enterprises/${androidEnterpriseId}/policies/${policyId}`,
            requestBody: { packageNames },
          });
          return response.data;
        }

        default: {
          let response = await androidManagementConnection.enterprises.policies.modifyPolicyApplications({
            name: `enterprises/${androidEnterpriseId}/policies/${policyId}`,
            requestBody: { changes },
          });
          return response.data;
        }
      }
    }).intercept({ status: 429 }, (err) => {
      // If the Android management API returns a 429 response, log an additional warning that will trigger a help-p1 alert.
      sails.log.warn(`p1: Android management API rate limit exceeded!`);
      return new Error(`When attempting to update applications for a policy of Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    }).intercept({status: 403}, ()=>{
      // If the Android management API returns a 403 response, return a enterpriseNotAccessible (notFound) response to the Fleet server.
      return {'enterpriseNotAccessible': 'Fleet is not authorized to manage this Android enterprise.'};
    }).intercept({ status: 404 }, (err) => {
      return {'policyNotFound': `Specified policy not found on this Android enterprise (${androidEnterpriseId}): ${err}`};
    }).intercept((err) => {
      if([502, 503, 504].includes(err.status)){
        return {'managementApiError': `The Android management API returned a transient 5xx error: ${err}`};
      }
      return new Error(`When attempting to update applications for a policy of Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${require('util').inspect(err)}`);
    });


    // Return the modified policy back to the Fleet server.
    return modifyApplicationPolicyResponse;

  }


};

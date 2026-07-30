module.exports = {


  friendlyName: 'Create android enrollment token',


  description: 'Creates and returns an enrollment token for an Android enterprise',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'An Android Enterprise enrollment token was returned to the Fleet instance.'},
    missingAuthHeader: { description: 'This request was missing an authorization header.', responseType: 'unauthorized'},
    unauthorized: { description: 'Invalid authentication token', responseType: 'unauthorized'},
    notFound: { description: 'No Android enterprise found for this Fleet server.', responseType: 'notFound'},
    enterpriseNotAccessible: { description: 'Fleet is not authorized to manage this Android enterprise.', responseType: 'notFound' },
    managementApiError: { statusCode: 503, description: 'The Android management API returned a transient 5xx error.' },
  },


  fn: async function ({androidEnterpriseId}) {
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
      androidEnterpriseId: androidEnterpriseId,
    });

    // Return a 404 response if no records are found.
    if(!thisAndroidEnterprise) {
      throw 'notFound';
    }

    // Return an unauthorized response if the provided secret does not match.
    if(thisAndroidEnterprise.fleetServerSecret !== fleetServerSecret) {
      throw 'unauthorized';
    }

    // Get the shared Google API auth client with the getAndroidManagementAuthorizationClient helper.
    // Note: we are doing this outside of the sails.helpers.flow.build() so any errors related to the website's credentials returned by the helper are not intercepted.
    let androidManagementAuthClient = await sails.helpers.androidProxy.getAndroidManagementAuthorizationClient();

    let newEnrollmentToken = await sails.helpers.flow.build(async ()=>{
      let { google } = require('googleapis');
      let androidManagementConnection = google.androidmanagement({version: 'v1', auth: androidManagementAuthClient});
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises$Enrollmenttokens.html#create
      let enrollmentTokenCreateResponse = await androidManagementConnection.enterprises.enrollmentTokens.create({
        parent: `enterprises/${androidEnterpriseId}`,
        // Note: Typically, we use defined inputs instead of accessing req.body directly. This behavior should not be repeated in future Android proxy endpoints.
        requestBody: this.req.body,
      });
      return enrollmentTokenCreateResponse.data;
    }).intercept({status: 429}, (err)=>{
      // If the Android management API returns a 429 response, log an additional warning that will trigger a help-p1 alert.
      sails.log.warn(`p1: Android management API rate limit exceeded!`);
      return new Error(`When attempting to create an enrollment token for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    }).intercept({status: 403}, ()=>{
      // If the Android management API returns a 403 response, return a enterpriseNotAccessible (notFound) response to the Fleet server.
      return {'enterpriseNotAccessible': 'Fleet is not authorized to manage this Android enterprise.'};
    }).intercept((err)=>{
      if([502, 503, 504].includes(err.status)) {
        return {'managementApiError': `The Android management API returned a transient 5xx error: ${err}`};
      }
      return new Error(`When attempting to create an enrollment token for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${require('util').inspect(err)}`);
    });


    return newEnrollmentToken;

  }


};

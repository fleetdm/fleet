module.exports = {


  friendlyName: 'Delete android device',


  description: 'Deletes a device of an Android enterprise',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    deviceId: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'The device of an Android enterprise was successfully deleted.' },
    missingAuthHeader: { description: 'This request was missing an authorization header.', responseType: 'unauthorized'},
    unauthorized: { description: 'Invalid authentication token.', responseType: 'unauthorized'},
    notFound: { description: 'No Android enterprise found for this Fleet server.', responseType: 'notFound'},
    enterpriseNotAccessible: { description: 'Fleet is not authorized to manage this Android enterprise.', responseType: 'notFound' },
    deviceNoLongerManaged: { description: 'The specified device is no longer managed by the Android enterprise.', responseType: 'notFound' },
    managementApiError: { statusCode: 503, description: 'The Android management API returned a transient 5xx error.' },
  },


  fn: async function ({ androidEnterpriseId, deviceId}) {

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

    // Delete the device for this Android enterprise.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occur using google's node library.
    await sails.helpers.flow.build(async () => {
      let { google } = require('googleapis');
      let androidManagementConnection = google.androidmanagement({version: 'v1', auth: androidManagementAuthClient});
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises$Devices.html#delete
      await androidManagementConnection.enterprises.devices.delete({
        name: `enterprises/${androidEnterpriseId}/devices/${deviceId}`,
      });
    }).intercept({status: 429}, (err)=>{
      // If the Android management API returns a 429 response, log an additional warning that will trigger a help-p1 alert.
      sails.log.warn(`p1: Android management API rate limit exceeded!`);
      return new Error(`When attempting to delete a device for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    }).intercept({status: 403}, ()=>{
      // If the Android management API returns a 403 response, return a enterpriseNotAccessible (notFound) response to the Fleet server.
      return {'enterpriseNotAccessible': 'Fleet is not authorized to manage this Android enterprise.'};
    }).intercept((err)=>{
      let errorString = err.toString();
      if (errorString.includes('Device is no longer being managed')) {
        return {'deviceNoLongerManaged': 'The device is no longer managed by the Android enterprise.'};
      }
      if([502, 503, 504].includes(err.status)){
        return {'managementApiError': `The Android management API returned a transient 5xx error: ${err}`};
      }
      return new Error(`When attempting to delete a device for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${require('util').inspect(err)}`);
    });


    // Return success response to the Fleet server.
    return {};

  }


};

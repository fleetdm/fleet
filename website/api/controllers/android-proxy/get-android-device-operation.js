module.exports = {


  friendlyName: 'Get android device operation',


  description: 'Gets a long-running operation for a device of an Android enterprise. Fleet servers poll this to recover the outcome of an Android MDM command (Lock, Wipe, Clear passcode) whose Pub/Sub COMMAND notification never arrived.',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    deviceId: {
      type: 'string',
      required: true,
    },
    operationId: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'The operation for a device of an Android enterprise was successfully retrieved.' },
    missingAuthHeader: { description: 'This request was missing an authorization header.', responseType: 'unauthorized'},
    unauthorized: { description: 'Invalid authentication token.', responseType: 'unauthorized'},
    notFound: { description: 'No Android enterprise found for this Fleet server.', responseType: 'notFound' },
    enterpriseNotAccessible: { description: 'Fleet is not authorized to manage this Android enterprise.', responseType: 'notFound' },
    operationNotFound: { description: 'The specified operation does not exist in this Android enterprise', responseType: 'notFound' },
  },


  fn: async function ({ androidEnterpriseId, deviceId, operationId }) {

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

    // Get the operation for this device.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occur using google's node library.
    let getOperationResponse = await sails.helpers.flow.build(async () => {
      let { google } = require('googleapis');
      let androidManagementConnection = google.androidmanagement({version: 'v1', auth: androidManagementAuthClient});
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises$Devices$Operations.html#get
      let getOperationResult = await androidManagementConnection.enterprises.devices.operations.get({
        name: `enterprises/${androidEnterpriseId}/devices/${deviceId}/operations/${operationId}`,
      });
      return getOperationResult.data;
    }).intercept({status: 429}, (err)=>{
      // If the Android management API returns a 429 response, log an additional warning that will trigger a help-p1 alert.
      sails.log.warn(`p1: Android management API rate limit exceeded!`);
      return new Error(`When attempting to get a device operation for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    }).intercept({status: 403}, ()=>{
      // If the Android management API returns a 403 response, return a enterpriseNotAccessible (notFound) response to the Fleet server.
      return {'enterpriseNotAccessible': 'Fleet is not authorized to manage this Android enterprise.'};
    }).intercept({status: 404}, ()=>{
      // If the Android management API returns a 404 response, return an operationNotFound (notFound) response to the Fleet server.
      // The Fleet server treats this as "Google no longer has a record of this command".
      return 'operationNotFound';
    }).intercept((err)=>{
      return new Error(`When attempting to get a device operation for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${require('util').inspect(err)}`);
    });


    // Return the operation data back to the Fleet server.
    return getOperationResponse;

  }


};

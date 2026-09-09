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
    // Unlike the other android-proxy actions, this one reserves 404 for a single meaning: the Android
    // management API has no record of this operation. The Fleet server treats that as evidence the
    // command can never complete and eventually marks it failed, so nothing about *this website's*
    // records may return a 404. A missing AndroidEnterprise row and a loss of access to the enterprise
    // both mean "we cannot answer for this enterprise" -- a 403, which the Fleet server classifies as
    // an authorization failure and stops its whole reconciler run on.
    enterpriseNotAccessible: { description: 'No Android enterprise found for this Fleet server, or Fleet is not authorized to manage it.', statusCode: 403 },
    operationNotFound: { description: 'The specified operation does not exist in this Android enterprise', responseType: 'notFound' },
    // The Fleet server classifies this status code to know it should stop polling and wait for the next
    // reconciler run, so the Android management API's 429 has to survive the trip through this proxy.
    tooManyRequests: { description: 'The Android management API rate limit was exceeded.', statusCode: 429 },
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

    // Return a 403 (not a 404) if no records are found -- see the note on the exits above.
    if (!thisAndroidEnterprise) {
      throw 'enterpriseNotAccessible';
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
      sails.androidProxyApiRequestCount++;// Count this Android Management API request toward the per-minute total logged in api/hooks/custom/index.js.
      sails.androidProxyApiRequestCountByEnterpriseId[androidEnterpriseId] = (sails.androidProxyApiRequestCountByEnterpriseId[androidEnterpriseId] || 0) + 1;// Count this request for the per-enterprise-per-minute total logged in api/hooks/custom/index.js.
      let getOperationResult = await androidManagementConnection.enterprises.devices.operations.get({
        name: `enterprises/${androidEnterpriseId}/devices/${deviceId}/operations/${operationId}`,
      });
      return getOperationResult.data;
    }).intercept({status: 429}, ()=>{
      // If the Android management API returns a 429 response, log an additional warning that will trigger a help-p1 alert.
      // Note: the error object is deliberately left out of this log -- gaxios errors carry the request
      // config, including the Authorization header used to call Google.
      sails.log.warn(`p1: Android management API rate limit exceeded! (When getting a device operation for Android enterprise ${androidEnterpriseId}.)`);
      // Pass the 429 through to the Fleet server rather than collapsing it into a 500, so its reconciler
      // can tell rate limiting apart from a generic failure.
      return 'tooManyRequests';
    }).intercept({status: 403}, ()=>{
      // If the Android management API returns a 403 response, return an enterpriseNotAccessible (403) response to the Fleet server.
      return 'enterpriseNotAccessible';
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

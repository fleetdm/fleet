module.exports = {


  friendlyName: 'Get android enterprise applications',


  description: 'Gets an android enterprise application',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    applicationId: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'The device of an Android enterprise was successfully retrieved.adfasd' },
    missingAuthHeader: { description: 'This request was missing an authorization header.', responseType: 'unauthorized'},
    unauthorized: { description: 'Invalid authentication token.', responseType: 'unauthorized'},
    notFound: { description: 'App not found', responseType: 'notFound' },
    deviceNoLongerManaged: { description: 'The device is no longer managed by the Android enterprise.', responseType: 'notFound' },
  },


  fn: async function ({ androidEnterpriseId, applicationId}) {

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

    // Get the device for this Android enterprise.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occur using google's node library.
    let getApplicationsResponse = await sails.helpers.flow.build(async () => {
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
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises$Applications.html#get
      let getApplicationsResult = await androidmanagement.enterprises.devices.get({
        name: `enterprises/${androidEnterpriseId}/applications/${applicationId}`,
      });
      return getApplicationsResult.data;
    }).intercept((err) => {
      let errorString = err.toString();
      if (errorString.includes('Device is no longer being managed')) {
        return {'deviceNoLongerManaged': 'The device is no longer managed by the Android enterprise.'};
      }
      return new Error(`When attempting to get an application for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    });


    // Return the device data back to the Fleet server.
    return getApplicationsResponse;

  }


};

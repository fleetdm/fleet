module.exports = {


  friendlyName: 'Get android devices',


  description: 'List android devices accessible to the Android enterprise.',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    pageSize: {
      type: 'number',
      description: 'The maximum number of devices to return.',
      min: 1,
      defaultsTo: 100,
      isInteger: true,
    },
    pageToken: {
      type: 'string',
    },
    fields: {
      type: 'string',
      description: 'Selector specifying which fields to include in a partial response if any.',
    }
  },


  exits: {
    success: { description: 'Android devices list was successfully retrieved.' },
    missingAuthHeader: { description: 'This request was missing an authorization header.', responseType: 'unauthorized'},
    missingOriginHeader: { description: 'The request was missing an Origin header', responseType: 'badRequest'},
    notFound: { description: 'No Android enterprise found for this Fleet server.', responseType: 'notFound' },
    unauthorized: { description: 'Invalid authentication token.', responseType: 'unauthorized'},
  },


  fn: async function ({ androidEnterpriseId, pageSize, pageToken, fields }) {

    // Extract fleetServerSecret from the Authorization header
    let authHeader = this.req.get('authorization');
    let fleetServerSecret;

    if (authHeader && authHeader.startsWith('Bearer')) {
      fleetServerSecret = authHeader.replace('Bearer', '').trim();
    } else {
      throw 'missingAuthHeader';
    }


    let thisAndroidEnterprise = await AndroidEnterprise.findOne({
      androidEnterpriseId: androidEnterpriseId
    });

    if (!thisAndroidEnterprise) {
      throw 'notFound';
    }

    if (thisAndroidEnterprise.fleetServerSecret !== fleetServerSecret) {
      throw 'unauthorized';
    }


    // List android devices from an enterprises using the passed parameters
    return await sails.helpers.flow.build(async ()=>{
      let { google } = require('googleapis');
      let androidmanagement = google.androidmanagement('v1');

      let googleAuth = new google.auth.GoogleAuth({
        scopes: [
          'https://www.googleapis.com/auth/androidmanagement'
        ],
        credentials: {
          client_email: sails.config.custom.androidEnterpriseServiceAccountEmailAddress,// eslint-disable-line camelcase
          private_key: sails.config.custom.androidEnterpriseServiceAccountPrivateKey,// eslint-disable-line camelcase
        },
      });

      // Acquire the google auth client
      let authClient = await googleAuth.getClient();
      google.options({auth: authClient});

      // Get the Android devices list from Google
      let devicesResponse = await androidmanagement.enterprises.devices.list({
        parent: `enterprises/${thisAndroidEnterprise.androidEnterpriseId}`,
        pageSize: pageSize,
        pageToken: pageToken,
        fields: fields,
      });
      // Return the devices (no filtering needed since parent parameter already filters by enterprise)
      return devicesResponse.data;

    }).intercept({status: 429}, (err)=>{
      // If the Android management API returns a 429 response, log an additional warning that will trigger a help-p1 alert.
      sails.log.warn(`p1: Android management API rate limit exceeded!`);
      return new Error(`When attempting to list devices for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    }).intercept((err)=>{
      return new Error(`When attempting to list devices for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    });
  }
};

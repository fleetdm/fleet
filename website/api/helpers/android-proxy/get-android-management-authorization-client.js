module.exports = {


  friendlyName: 'Get Android Management authorization client',


  description: 'Returns a shared Google API auth client for the Android Management API proxy, creating it if one has not been created yet.',


  moreInfoUrl: 'https://github.com/fleetdm/fleet/issues/46496',


  exits: {

    success: {
      outputFriendlyName: 'Android Management authorization client',
      outputDescription: 'The shared Google API auth client stored on `sails.androidManagementAuthorization`.',
      outputType: 'ref',
    },
  },


  fn: async function () {

    require('assert')(sails.config.custom.androidEnterpriseServiceAccountEmailAddress);
    require('assert')(sails.config.custom.androidEnterpriseServiceAccountPrivateKey);

    // Initialize a Google API auth client for the Android Management API proxy, but only if one has not
    // already been created for this server process. The googleapis library caches the OAuth2 access_token
    // on a reused client and refreshes it automatically when it expires, so we build a single shared client
    // per process (each web dyno is its own process) and reuse it across all Android proxy requests.
    if (!sails.androidManagementAuthorization) {
      let { google } = require('googleapis');
      let googleAuth = new google.auth.GoogleAuth({
        // The pubsub scope is included because creating/deleting an Android enterprise also provisions/removes a Pub/Sub topic and subscription.
        scopes: [
          'https://www.googleapis.com/auth/androidmanagement',
          'https://www.googleapis.com/auth/pubsub',
        ],
        credentials: {
          client_email: sails.config.custom.androidEnterpriseServiceAccountEmailAddress,// eslint-disable-line camelcase
          private_key: sails.config.custom.androidEnterpriseServiceAccountPrivateKey,// eslint-disable-line camelcase
        },
      });
      let androidManagementAuthClient = await googleAuth.getClient();
      // Mint an access token now so invalid credentials surface here instead of failing silently on the
      // first Android Management API call. This only hits Google's OAuth2 token endpoint, not the Android
      // Management API, so it does not count against AMAPI rate limits.
      await androidManagementAuthClient.getAccessToken();
      // Assign the global last, so that if either step above throws, the global is left unset and the next
      // request retries instead of caching a client whose credentials never validated.
      sails.androidManagementAuthorization = androidManagementAuthClient;
    }

    return sails.androidManagementAuthorization;

  }


};


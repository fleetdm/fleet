module.exports = {


  friendlyName: 'Register one Android device',


  description: 'Exchanges a server auth code from the Fleet Android app for the end user\'s tokens and, when the app read the device\'s GSF ID, looks up the device in Google.',


  inputs: {
    googleCustomerId: {
      type: 'string',
      required: true,
    },
    fleetServerSecret: {
      type: 'string',
      required: true,
    },
    serverAuthCode: {
      type: 'string',
      description: 'A server auth code from Google Identity Services on the device.',
      required: true,
    },
    hostUuid: {
      type: 'string',
      required: true,
    },
    androidId: {
      type: 'string',
      description: 'The device\'s Google Services Framework ID (16 hex characters), when the Fleet app could read it.',
    },
  },


  exits: {
    success: { description: 'The signed-in user\'s email, and the device\'s DeviceUser names when an androidId was provided, were returned to the Fleet server.' },
    customerNotFound: { description: 'A Google conditional access registration could not be found using the provided information.', responseType: 'notFound' },
    googleRefusedRequest: { description: 'Google refused the code exchange or the lookup. The response body carries Google\'s error.', statusCode: 422 },
  },


  fn: async function ({googleCustomerId, fleetServerSecret, serverAuthCode, androidId}) {
    // TODO: hostUuid is required by the website spec and sent by the Fleet server (#56650), but nothing says what fleetdm.com does with it.

    // The Fleet server stores the customer ID without the leading "C" (#56649), and fleetdm.com stores it with one.
    googleCustomerId = 'C' + googleCustomerId.replace(/^C/, '');

    let informationAboutThisCustomer = await GoogleConditionalAccessServer.findOne({googleCustomerId, fleetServerSecret});
    if(!informationAboutThisCustomer) {
      throw 'customerNotFound';
    }

    // Exchange the server auth code for the end user's access token and ID token.
    // Note: Server auth codes from Android are exchanged with an empty redirect URI.
    let tokenResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: 'https://oauth2.googleapis.com/token',
      enctype: 'application/x-www-form-urlencoded',
      body: {
        code: serverAuthCode,
        client_id: sails.config.custom.googleConditionalAccessOauthClientId,// eslint-disable-line camelcase
        client_secret: sails.config.custom.googleConditionalAccessOauthClientSecret,// eslint-disable-line camelcase
        redirect_uri: '',// eslint-disable-line camelcase
        grant_type: 'authorization_code',// eslint-disable-line camelcase
      }
    }).intercept('non200Response', (err)=>{
      let googleError;
      try {
        googleError = JSON.parse(err.raw.body);
      } catch(unusedErr) {
        googleError = {error: err.raw.body};
      }
      return {googleRefusedRequest: googleError};
    }).intercept((err)=>{
      return new Error(`When exchanging a server auth code from an Android device for a Google conditional access customer (${googleCustomerId}), an error occurred. Full error: ${require('util').inspect(err, {depth: 3})}`);
    });

    let accessToken;
    let email;
    try {
      let parsedTokenResponse = JSON.parse(tokenResponse.body);
      accessToken = parsedTokenResponse.access_token;
      // Note: The ID token comes straight from Google's token endpoint, so its signature doesn't need to be verified.
      email = JSON.parse(Buffer.from(parsedTokenResponse.id_token.split('.')[1], 'base64url').toString()).email;
    } catch(err) {
      throw new Error(`When parsing the response from Google's token endpoint for a Google conditional access customer (${googleCustomerId}), an error occurred. Full error: ${require('util').inspect(err)}`);
    }
    if(!accessToken || !email) {
      throw new Error(`When exchanging a server auth code from an Android device for a Google conditional access customer (${googleCustomerId}), Google's token endpoint did not return an access token and an ID token with an email.`);
    }

    // Without an androidId, the Fleet server finds the device through the customer's device listing itself (#57698).
    if(!androidId) {
      return {
        email: email,
      };
    }

    // [?]: https://cloud.google.com/identity/docs/reference/rest/v1/devices.deviceUsers/lookup
    let lookupResponse = await sails.helpers.http.get.with({
      url: 'https://cloudidentity.googleapis.com/v1/devices/-/deviceUsers:lookup',
      data: {
        userId: 'me',
        androidId: androidId,
      },
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    }).intercept('non200Response', (err)=>{
      let googleError;
      try {
        googleError = JSON.parse(err.raw.body);
      } catch(unusedErr) {
        googleError = {error: err.raw.body};
      }
      return {googleRefusedRequest: googleError};
    }).intercept((err)=>{
      return new Error(`When looking up an Android device for a Google conditional access customer (${googleCustomerId}), an error occurred. Full error: ${require('util').inspect(err, {depth: 3})}`);
    });

    // An empty list of names means Google has no record for this androidId.
    return {
      names: lookupResponse.names || [],
      customer: lookupResponse.customer,
      email: email,
    };

  }


};

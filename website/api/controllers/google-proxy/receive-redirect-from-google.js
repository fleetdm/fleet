module.exports = {


  friendlyName: 'Receive redirect from Google',


  description: 'Looks up an iOS/iPadOS device in Google after the end user signs in, sends the DeviceUser names to the Fleet server, and redirects to a result page.',


  inputs: {
    code: {
      type: 'string',
      description: 'The authorization code from Google sign-in.',
    },
    state: {
      type: 'string',
      description: 'The values from the start sign-in request, carried through Google sign-in.',
      required: true,
    },
    error: {
      type: 'string',
      description: 'This value is only present if sign-in failed or the end user cancelled it.',
    },
  },


  exits: {
    success: { description: 'The end user was redirected to a result page.', responseType: 'redirect' },
    redirect: { responseType: 'redirect' },
    fleetInstanceNotOnAllowlist: {description: 'This Fleet instance is not on the allowlist.', responseType: 'unauthorized' },
  },


  fn: async function ({code, state, error}) {

    let errorPageUrl = '/google-conditional-access/error';

    if(error || !code) {
      return errorPageUrl;
    }

    let valuesFromSignIn;
    try {
      valuesFromSignIn = JSON.parse(Buffer.from(state, 'base64url').toString());
    } catch(unusedErr) {
      return errorPageUrl;
    }
    if(!_.isObject(valuesFromSignIn) || !_.isString(valuesFromSignIn.googleCustomerId) || !_.isString(valuesFromSignIn.token) || !_.isString(valuesFromSignIn.hostUuid)) {
      return errorPageUrl;
    }

    let informationAboutThisCustomer = await GoogleConditionalAccessServer.findOne({googleCustomerId: valuesFromSignIn.googleCustomerId});
    if(!informationAboutThisCustomer) {
      return errorPageUrl;
    }
    // Return a fleetInstanceNotOnAllowlist response if this server url has not been added to sails.config.custom.googleConditionalAccessAllowedOrigins
    if(!sails.config.custom.googleConditionalAccessAllowedOrigins.includes(informationAboutThisCustomer.fleetInstanceUrl)) {
      throw 'fleetInstanceNotOnAllowlist';
    }

    // Exchange the authorization code for the end user's access token and ID token.
    let tokenResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: 'https://oauth2.googleapis.com/token',
      enctype: 'application/x-www-form-urlencoded',
      body: {
        code: code,
        client_id: sails.config.custom.googleConditionalAccessOauthClientId,// eslint-disable-line camelcase
        client_secret: sails.config.custom.googleConditionalAccessOauthClientSecret,// eslint-disable-line camelcase
        redirect_uri: `${sails.config.custom.baseUrl}/api/v1/google-conditional-access/callback`,// eslint-disable-line camelcase
        grant_type: 'authorization_code',// eslint-disable-line camelcase
      }
    }).intercept((err)=>{
      sails.log.warn(`When exchanging an authorization code for a Google conditional access customer (${informationAboutThisCustomer.googleCustomerId}), an error occurred. Full error: ${require('util').inspect(err, {depth: 3})}`);
      return {redirect: errorPageUrl};
    });

    let accessToken;
    let email;
    try {
      let parsedTokenResponse = JSON.parse(tokenResponse.body);
      accessToken = parsedTokenResponse.access_token;
      // Note: The ID token comes straight from Google's token endpoint, so its signature doesn't need to be verified.
      email = JSON.parse(Buffer.from(parsedTokenResponse.id_token.split('.')[1], 'base64url').toString()).email;
    } catch(err) {
      sails.log.warn(`When parsing the response from Google's token endpoint for a Google conditional access customer (${informationAboutThisCustomer.googleCustomerId}), an error occurred. Full error: ${require('util').inspect(err)}`);
      return errorPageUrl;
    }
    if(!accessToken || !email) {
      return errorPageUrl;
    }

    // [?]: https://cloud.google.com/identity/docs/reference/rest/v1/devices.deviceUsers/lookup
    let lookupResponse = await sails.helpers.http.get.with({
      url: 'https://cloudidentity.googleapis.com/v1/devices/-/deviceUsers:lookup',
      data: {
        userId: 'me',
        partner: sails.config.custom.googleConditionalAccessPartnerId,
        iosDeviceId: valuesFromSignIn.hostUuid,
      },
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    }).intercept((err)=>{
      sails.log.warn(`When looking up an iOS/iPadOS device for a Google conditional access customer (${informationAboutThisCustomer.googleCustomerId}), an error occurred. Full error: ${require('util').inspect(err, {depth: 3})}`);
      return {redirect: errorPageUrl};
    });

    // Google returns the customer as a resource name (e.g. "customers/01234abc"), and the Fleet server expects the ID with the leading "C".
    let customer;
    if(_.isString(lookupResponse.customer)) {
      customer = 'C' + lookupResponse.customer.replace(/^customers\//, '').replace(/^C/, '');
    }

    // Google returns no names until the user signs in to Gmail on this device. An empty list is sent as-is, and the
    // Fleet server answers with `not_found`.
    let registerResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'POST',
      url: `${informationAboutThisCustomer.fleetInstanceUrl}/api/v1/fleet/google_conditional_access/register`,
      headers: {
        'Authorization': `Bearer ${informationAboutThisCustomer.fleetServerSecret}`
      },
      body: {
        token: valuesFromSignIn.token,
        names: lookupResponse.names || [],
        customer: customer,
        email: email,
      }
    }).intercept((err)=>{
      sails.log.warn(`When sending a device's DeviceUser names to a Fleet instance (${informationAboutThisCustomer.fleetInstanceUrl}) for a Google conditional access customer, an error occurred. Full error: ${require('util').inspect(err, {depth: 3})}`);
      return {redirect: errorPageUrl};
    });

    let registrationResult;
    try {
      registrationResult = JSON.parse(registerResponse.body).result;
    } catch(unusedErr) {
      return errorPageUrl;
    }

    if(registrationResult === 'done') {
      return '/google-conditional-access/done';
    } else if(registrationResult === 'not_allowed') {
      return '/google-conditional-access/not-allowed';
    } else if(registrationResult === 'not_found') {
      return '/google-conditional-access/not-found';
    } else {
      return errorPageUrl;
    }

  }


};

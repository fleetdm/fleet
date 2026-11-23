module.exports = {


  friendlyName: 'Redirect to Google sign-in',


  description: 'Redirects an end user on an iOS/iPadOS host to Google sign-in with Fleet\'s OAuth client.',


  inputs: {
    googleCustomerId: {
      type: 'string',
      required: true,
    },
    token: {
      type: 'string',
      description: 'A short-lived, single-use token that the Fleet server issued for this host.',
      required: true,
    },
    hostUuid: {
      type: 'string',
      description: 'The host\'s UUID. Sent to Google as the iosDeviceId on lookup.',
      required: true,
    },
  },


  exits: {
    success: { description: 'The end user was redirected to Google sign-in.', responseType: 'redirect' },
    customerNotFound: { description: 'The provided Google customer ID has no Google conditional access registration.', responseType: 'notFound' },
    fleetInstanceNotOnAllowlist: {description: 'This Fleet instance is not on the allowlist.', responseType: 'unauthorized' },
  },


  fn: async function ({googleCustomerId, token, hostUuid}) {

    // Google's device APIs use the customer ID without the leading "C", so accept either form and match it with one.
    googleCustomerId = 'C' + googleCustomerId.replace(/^C/, '');

    let informationAboutThisCustomer = await GoogleConditionalAccessServer.findOne({googleCustomerId});
    if(!informationAboutThisCustomer) {
      throw 'customerNotFound';
    }
    // Return a fleetInstanceNotOnAllowlist response if this server url has not been added to sails.config.custom.googleConditionalAccessAllowedOrigins
    if(!sails.config.custom.googleConditionalAccessAllowedOrigins.includes(fleetInstanceUrl)) {
      throw 'fleetInstanceNotOnAllowlist';
    }

    let state = Buffer.from(JSON.stringify({googleCustomerId, token, hostUuid})).toString('base64url');

    let googleSignInParams = new URLSearchParams({
      client_id: sails.config.custom.googleConditionalAccessOauthClientId,// eslint-disable-line camelcase
      redirect_uri: `${sails.config.custom.baseUrl}/api/v1/google-conditional-access/callback`,// eslint-disable-line camelcase
      response_type: 'code',// eslint-disable-line camelcase
      scope: 'openid email https://www.googleapis.com/auth/cloud-identity.devices.lookup',
      state: state,
    });

    return `https://accounts.google.com/o/oauth2/v2/auth?${googleSignInParams.toString()}`;

  }


};

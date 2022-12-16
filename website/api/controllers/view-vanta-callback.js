module.exports = {


  friendlyName: 'View vanta callback',


  description: 'Display "Vanta callback" page.',


  inputs: {
    state: {
      type: 'string',
    },
    code: {
      type: 'string',
    }
  },


  exits: {

    success: {
      viewTemplatePath: 'pages/vanta-callback'
    },

    stateDoesNotMatch: {
      description: 'The requesting user\'s state cookie could not be matched with the query parameters set by Vanta',
      responseType: 'redirect',
    },

    missingCookies: {
      description: 'The requesting user is missing cookies required to verify their identity',
      responseType: 'redirect',
    },

    redirect: {
      description: 'No query parameters recieved from Vanta, the requesting user will be sent to the homepage.',
      responseType: 'redirect',
    },

    couldNotAuthorize: {
      description: 'Vanta returned a non-200 response when an authorization token was requested for this Vanta connection.',
      responseType: '400',
    }


  },


  fn: async function (inputs) {

    // If we're missing any query parameters sent from Vanta, redirect to the homepage.
    if(!inputs.state || !inputs.code) {
      throw {redirect: '/'};
    }
    // If query parameters were provided, but they don't match
    if(this.req.signedCookies.state !== inputs.state){
      throw {stateDoesNotMatch: '/'};
    }
    if(!this.req.signedCookies.oauthSourceIdForFleet || !this.req.signedCookies.state){
      throw {missingCookies: '/'};
    }

    let recordOfThisAuthorization = await VantaConnection.findOne({vantaSourceId: this.req.signedCookies.oauthSourceIdForFleet});

    if(!recordOfThisAuthorization){
      throw new Error(`When a user tried to connect their Vanta account with their Fleet instance, the VantaConnection record associated with the request could not be found.`);
    }

    let vantaAuthorizationResponse = await sails.helpers.http.post(
      'https://api.vanta.com/oauth/token',
      {
        'client_id': sails.config.custom.vantaAuthorizationClientId,
        'client_secret': sails.config.custom.vantaAuthorizationClientSecret,
        'code': inputs.code,
        'redirect_uri': sails.config.custom.baseUrl+'/vanta-callback',
        'source_id': recordOfThisAuthorization.vantaSourceId,
        'grant_type': 'authorization_code',
      }
    ).catch(()=>{
      throw 'couldNotAuthorize';
    });

    await VantaConnection.updateOne({id: recordOfThisAuthorization.id}).set({
      vantaToken: vantaAuthorizationResponse.access_token,
      vantaTokenExpiresAt: Date.now() + (vantaAuthorizationResponse.expires_in * 1000),
      vantaRefreshToken: vantaAuthorizationResponse.refresh_token,
      isConnectedToVanta: true,
    });

    return {
      showSuccessMessage: true
    };

  }


};

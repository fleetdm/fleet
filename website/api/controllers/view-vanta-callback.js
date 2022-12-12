module.exports = {


  friendlyName: 'View vanta callback',


  description: 'Display "Vanta callback" page.',


  inputs: {
    state: {
      type: 'string',
      required: true
    },
    code: {
      type: 'string',
      required: true,
    }
  },


  exits: {

    success: {
      viewTemplatePath: 'pages/vanta-callback'
    },

    redirect: {
      description: 'The requesting user',
      responseType: 'redirect',
    },


  },


  fn: async function (inputs) {

    if(this.req.signedCookies.state !== inputs.state){
      sails.log('mismatched state :o');
      throw {redirect: '/'};
    }

    if(!this.req.signedCookies.oauthSourceIdForFleet){
      sails.log('missing source id cookie!');
      throw {redirect: '/'};
    }

    let recordOfThisAuthorization = await VantaConnection.findOne({emailAddress: this.req.signedCookies.oauthSourceIdForFleet});
    // console.log(inputs);

    let vantaAuthorizationResponse = await sails.helpers.http.post(
      'https://api.vanta.com/oauth/token',
      {
        'client_id': sails.config.custom.vantaAuthorizationClientId,
        'client_secret': sails.config.custom.vantaAuthorizationClientSecret,
        'code': inputs.code,
        'redirect_uri': sails.config.custom.baseUrl+'/vanta-callback',
        'source_id': recordOfThisAuthorization.emailAddress,
        'grant_type': 'authorization_code',
      }
    );

    await VantaConnection.updateOne({id: recordOfThisAuthorization.id}).set({
      authToken: vantaAuthorizationResponse.access_token,
      authTokenExpiresAt: Date.now() + (vantaAuthorizationResponse.expires_in * 1000),
      refreshToken: vantaAuthorizationResponse.refresh_token,
      isConnectedToVanta: true,
    });

    return {
      showSuccessMessage: true
    };

  }


};

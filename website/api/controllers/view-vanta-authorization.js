module.exports = {


  friendlyName: 'View vanta authorization',


  description: 'Display "Vanta authorization" page.',


  inputs: {
    state: {
      type: 'string',
      description: 'The state provided to Vanta when an authorization request was created',
      required: true,
    },

    code: {
      type: 'string',
      description: 'The parameter that will be exchanged for a Vanta authorization token for this request.',
      required: true,
    }
  },


  exits: {

    success: {
      viewTemplatePath: 'pages/vanta-authorization',
    },

    redirect: {
      description: 'The requesting user will be redirected to the URL they specified after set up.',
      responseType: 'redirect'
    },

  },


  fn: async function (inputs) {

    // If either of the required cookies are missing, but a state and code were provided via query string. we'll show an error to the user, and they will be asked to try the authorization flow again.
    if(!this.req.signedCookies.vantaSourceId || !this.req.signedCookies.state){
      return {
        showSuccessMessage: false,
        connectionError: 'missingCookies',
      };
    }

    // If the provided state doesn't match the `state` cookie set for this authorization request. We'll display an error to the user
    if(this.req.signedCookies.state !== inputs.state) {
      return {
        showSuccessMessage: false,
        connectionError: 'mismatchedState',
      };
    }

    // Find the VantaConnection record that we created when the user created this request.
    let recordOfThisAuthorization = await VantaConnection.findOne({vantaSourceId: this.req.signedCookies.vantaSourceId});

    if(!recordOfThisAuthorization){
      // If no record of this authorization could be found, but the user has a `state` and `vantaSourceId` cookie, throw an error.
      throw new Error(`When a user tried to connect their Vanta account with their Fleet instance. No VantaConnection record with the sourceID ${this.req.signedCookies.vantaSourceId} could be found.`);
    }
    if(recordOfThisAuthorization.isConnectedToVanta) {
      return {
        showSuccessMessage: true,
      };
    }

    // Send an authorization request to Vanta,
    let vantaAuthorizationResponse = await sails.helpers.http.post(
      'https://api.vanta.com/oauth/token',
      {
        'client_id': sails.config.custom.vantaAuthorizationClientId,
        'client_secret': sails.config.custom.vantaAuthorizationClientSecret,
        'code': inputs.code,
        'redirect_uri': sails.config.custom.baseUrl+'/vanta-authorization',
        'source_id': recordOfThisAuthorization.vantaSourceId,
        'grant_type': 'authorization_code',
      }
    ).intercept((error)=>{// If an error occurs while sending an authorization request, throw an error.
      return new Error(`When requesting an authorization token from Vanta for a Vanta connection with id ${recordOfThisAuthorization.id}, an error occurred. Full error: ${error}`);
    });

    // Update the VantaConnection record for this request with information from the authorization response from Vanta.
    let updatedRecord = await VantaConnection.updateOne({id: recordOfThisAuthorization.id}).set({
      vantaAuthToken: vantaAuthorizationResponse.access_token,
      vantaAuthTokenExpiresAt: Date.now() + (vantaAuthorizationResponse.expires_in * 1000), // The expires_in value in the response from Vanta is the number of seconds until the access_token expires, so we'll create a new JS timestamp set to that time.
      vantaRefreshToken: vantaAuthorizationResponse.refresh_token,
      isConnectedToVanta: true,
    });

    if(!updatedRecord){
      throw new Error(`When trying to update a VantaConnection record (id: ${recordOfThisAuthorization.id}) with an authorization token from Vanta, the database record associated with this request has gone missing.`);
    }
    if(this.req.signedCookies.redirectAfterSetup){
      let redirectUrl = this.req.signedCookies.redirectAfterSetup;
      throw {redirect: redirectUrl};
    }

    return {
      showSuccessMessage: true
    };

  }


};

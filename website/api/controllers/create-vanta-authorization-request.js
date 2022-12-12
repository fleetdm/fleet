module.exports = {


  friendlyName: 'Get vanta authorization request',


  description: '',


  inputs: {
    emailAddress: {
      type: 'string',
      required: true,
    },
    fleetInstanceUrl: {
      type: 'string',
      required: true,
    },
    fleetApiKey: {
      type: 'string',
      required: true,
    }
  },


  exits: {
    emailAlreadyInUse: {
      statusCode: 409,
      description: 'The provided email address is already in use.',
    },
    fleetInstanceNotResponding: {
      description: 'The Fleet instance did not respond.',
    },
    invalidToken: {
      description: 'The provided token for the api-only user could not be used to authorize requests from fleetdm.com'
    },
    nonApiOnlyUser: {
      description: 'The provided API token for this Fleet instance is not associated with an api-only user.'
    },
    insufficientPermissions:{
      description: 'The api-only user associated with the provided token does not have the propper permissions to query the users endpoint.'
    },

  },


  fn: async function (inputs) {

    let generatedStateForThisRequest = await sails.helpers.strings.random.with({len: 10});

    // Validate the URL provided.
    let urlWithProtocol = inputs.fleetInstanceUrl;
    if(!_.startsWith(inputs.fleetInstanceUrl, 'https://') && !_.startsWith(inputs.fleetInstanceUrl, 'http://')){
      urlWithProtocol = 'https://' + urlWithProtocol;
    }

    // Check the fleet instance url and API key provided
    let responseFromFleetInstance = await sails.helpers.http.get(urlWithProtocol+'/api/v1/fleet/me',{},{'Authorization': 'Bearer ' +inputs.fleetApiKey})
    .intercept('requestFailed','fleetInstanceNotResponding')
    .intercept('non200Response', 'invalidToken');

    if(!responseFromFleetInstance.user.api_only){
      throw 'nonApiOnlyUser';
    }

    if(responseFromFleetInstance.user.global_role !== 'admin'){
      throw 'insufficientPermissions';
    }

    let connectionRecord = await VantaConnection.create({
      emailAddress: inputs.emailAddress,
      fleetInstanceUrl: urlWithProtocol,
      fleetApiKey: inputs.fleetApiKey,
    });

    let vantaAuthorizationRequestURL = `https://app.vanta.com/oauth/authorize?client_id=${sails.config.custom.vantaAuthorizationClientId}&scope=connectors.self:write-resource connectors.self:read-resource&state=${generatedStateForThisRequest}&source_id=${encodeURIComponent(inputs.emailAddress)}&redirect_uri=${url.resolve(sails.config.custom.baseUrl, '/vanta-callback')}&response_type=code`;

    // Set a `state` cookie on the user's browser. This value will be checked against a query parameter when the user returns to fleetdm.com.
    this.res.cookie('state', generatedStateForThisRequest, {signed: true});

    // Set the user's email address to a cookie, we'll use this value to find the database record we created for this Vanta Connection when the user returns to fleetdm.com.
    this.res.cookie('oauthSourceIdForFleet', inputs.emailAddress, {signed: true});

    return vantaAuthorizationRequestURL;
  }


};

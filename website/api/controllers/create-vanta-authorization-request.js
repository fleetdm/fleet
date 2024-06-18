module.exports = {


  friendlyName: 'Create Vanta authorization request',


  description: 'Returns a URL used to authorize requests to the user\'s Vanta account from fleetdm.com on behalf of the user.',


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
    },
    redirectToExternalPageAfterAuthorization: {
      type: 'string',
      description: 'If provided, the user will be sent to this URL after they complete the setup of this integration'
    },
    sharedSecret: {
      type: 'string',
      description: 'A shared secret used to verify external requests to this endpoint.',
      extendedDescription: 'This input is used only when this action runs at the "/api/v1/create-external-vanta-authorization-request" endpoint'
    }
  },


  exits: {
    success: {
      outputType: 'string'
    },
    connectionAlreadyExists: {
      description: 'The Fleet instance url provided is already connected to a Vanta account.',
      statusCode: 409,
    },
    fleetInstanceNotResponding: {
      description: 'A http request to the user\'s Fleet instance failed.',
      statusCode: 404,
    },
    invalidToken: {
      description: 'The provided token for the api-only user could not be used to authorize requests from fleetdm.com',
      statusCode: 403,
    },
    invalidLicense: {
      description: 'The Fleet instance provided is using a Free license.',
      statusCode: 400,
    },
    invalidResponseFromFleetInstance: {
      description: 'The response body from the Fleet API was invalid.',
      statusCode: 400,
    },
    nonApiOnlyUser: {
      description: 'The provided API token for this Fleet instance is not associated with an api-only user.',
      statusCode: 400,
    },
    insufficientPermissions:{
      description: 'The api-only user associated with the provided token does not have the propper permissions to query the users endpoint.',
      statusCode: 403,
    },
    missingOrInvalidSharedSecret: {
      description: 'The request to set up a Vanta integration has an invalid shared secret',
      statusCode: 401
    }
  },

  fn: async function (inputs) {
    require('assert')(sails.config.custom.sharedSecretForExternalVantaRequests);
    if(this.req.url === '/api/v1/create-external-vanta-authorization-request' && inputs.sharedSecret !== sails.config.custom.sharedSecretForExternalVantaRequests) {
      throw 'missingOrInvalidSharedSecret';
    }

    let url = require('url');

    // Look for any existing VantaConnection records that use this fleet instance URL.
    let existingConnectionRecord = await VantaConnection.findOne({fleetInstanceUrl: inputs.fleetInstanceUrl});

    // Generate the `state` string for this request.
    let generatedStateForThisRequest = await sails.helpers.strings.random.with({len: 10});

    // Generate a sourceId for this user. This value will be used as the indentifier of ther user's vanta connection
    let generatedSourceIdSuffix = await sails.helpers.strings.random.with({len: 20, style: 'url-friendly'});
    let sourceIDForThisRequest = 'fleet_'+generatedSourceIdSuffix;

    if(existingConnectionRecord) {
      // If an active Vanta connection exists for the provided Fleet instance url, we'll throw a 'connectionAlreadyExists' exit, and the user will be asked to contact us to make changes to the existing vanta connection.
      if(existingConnectionRecord.isConnectedToVanta) {
        throw 'connectionAlreadyExists';
      } else if(existingConnectionRecord.fleetApiKey !== inputs.fleetApiKey && existingConnectionRecord.emailAddress !== inputs.emailAddress) {
      // If an incomplete connection exists, and the API token and email address provided do not match. The user will be asked to contact us to make changes to their connection.
        throw 'connectionAlreadyExists';
      } else {
        // If an inactive and incomplete Vanta connection exists that uses the same API token and email address, we'll use the sourceId from that record for this request.
        sourceIDForThisRequest = existingConnectionRecord.vantaSourceId;
      }
    }


    // Check the fleet instance url and API key provided
    let responseFromFleetInstance = await sails.helpers.http.get(inputs.fleetInstanceUrl+'/api/v1/fleet/me',{},{'Authorization': 'Bearer ' +inputs.fleetApiKey})
    .intercept('requestFailed', 'fleetInstanceNotResponding')
    .intercept('non200Response', 'invalidToken')
    .intercept((error)=>{
      return new Error(`When sending a request to a Fleet instance's /me endpoint to verify that a token meets the requirements for a Vanta connection, an error occurred: ${error}`);
    });

    // Throw an error if the response from the Fleet instance's /me API endpoint does not contain a user.
    if(!responseFromFleetInstance.user){
      throw 'invalidResponseFromFleetInstance';
    }

    // Throw an error if the provided API token is not an API-only user.
    if(!responseFromFleetInstance.user.api_only) {
      throw 'nonApiOnlyUser';
    }

    // If the API-only user associated with the token provided does not have the admin role, we'll throw an error.
    // We require an admin token so we can send Vanta data about all of the active user accounts on the requesting user's Fleet instance
    if(responseFromFleetInstance.user.global_role !== 'admin') {
      throw 'insufficientPermissions';
    }

    // Send a request to the provided Fleet instance's /config endpoint to check their license tier.
    let configResponse = await sails.helpers.http.get(inputs.fleetInstanceUrl+'/api/v1/fleet/config', {}, {'Authorization': 'Bearer ' +inputs.fleetApiKey})
    .intercept('requestFailed','fleetInstanceNotResponding')
    .intercept('non200Response', 'invalidToken')
    .intercept((error)=>{
      return new Error(`When sending a request to a Fleet instance's /config API endpoint for a Vanta connection, an error occurred: ${error}`);
    });


    // Throw an error if the response from the Fleet instance's /config API endpoint does not contain a license.
    if(!configResponse.license){
      throw 'invalidResponseFromFleetInstance';
    }

    // If the user's Fleet instance has a free license, we'll throw the 'invalidLicense' exit and let the user know that this is only available for Fleet Premium subscribers.
    if(configResponse.license.tier === 'free') {
      throw 'invalidLicense';
    }

    // If we're not using an existing vantaConnection record for this request, we'll create a new one.
    if(!existingConnectionRecord) {
      // Create the VantaConnection record for this request.
      await VantaConnection.create({
        emailAddress: inputs.emailAddress,
        vantaSourceId: sourceIDForThisRequest,
        fleetInstanceUrl: inputs.fleetInstanceUrl,
        fleetApiKey: inputs.fleetApiKey,
      });
    }
    // Build the authorization URL for this request.
    let vantaAuthorizationRequestURL = `https://app.vanta.com/oauth/authorize?client_id=${encodeURIComponent(sails.config.custom.vantaAuthorizationClientId)}&scope=connectors.self:write-resource connectors.self:read-resource&state=${encodeURIComponent(generatedStateForThisRequest)}&source_id=${encodeURIComponent(sourceIDForThisRequest)}&redirect_uri=${encodeURIComponent(url.resolve(sails.config.custom.baseUrl, '/vanta-authorization'))}&response_type=code`;

    if(inputs.redirectToExternalPageAfterAuthorization){
      let internalRedirectUrl =  `${sails.config.custom.baseUrl}/redirect-vanta-authorization-request?vantaSourceId=${encodeURIComponent(sourceIDForThisRequest)}&state=${encodeURIComponent(generatedStateForThisRequest)}&vantaAuthorizationRequestURL=${encodeURIComponent(vantaAuthorizationRequestURL)}&redirectAfterSetup=${encodeURIComponent(inputs.redirectToExternalPageAfterAuthorization)}`;

      return internalRedirectUrl;
      // If the useInternalRedirect input was provided, we'll return the URL of an internal endpoiint that will set the required cookies for this request.
    } else {
      // Otherwise, if this request came from a user on the connect-vanta page, we'll set the cookies are redirect them directly to Vanta.
      // Set a `state` cookie on the user's browser. This value will be checked against a query parameter when the user returns to fleetdm.com.
      this.res.cookie('state', generatedStateForThisRequest, {signed: true});
      // Set the sourceId to a cookie, we'll use this value to find the database record we created for this request when the user returns to fleetdm.com.
      this.res.cookie('vantaSourceId', sourceIDForThisRequest, {signed: true});
      return vantaAuthorizationRequestURL;
    }
  }


};

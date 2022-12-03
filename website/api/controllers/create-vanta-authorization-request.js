module.exports = {


  friendlyName: 'Create vanta authorization request',


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
      statusCode: 404,
      description: 'The Fleet intance did not respond.',
    },
  },


  fn: async function (inputs) {

    // let generatedSourceIdForThisRequest = await sails.helpers.strings.random.with({len: 10});
    let generatedStateForThisRequest = await sails.helpers.strings.random.with({len: 10});

    if(await User.findOne({emailAddress: inputs.emailAddress})) {
      throw 'emailAlreadyInUse';
    }
    // Check the fleet instance url and API key provided
    let responseFromFleetInstance = await sails.helpers.http.get(inputs.fleetInstanceUrl+'/api/v1/fleet/me',{},{'Authorization': 'Bearer ' +inputs.fleetApiKey}).intercept(['requestFailed', 'non200Response'], (err)=>{
      // If we recieved a non-200 response from the cloud provisioner API, we'll throw a 500 error.
      return new Error('The fleet instance didnt like that');
    });
    console.log(responseFromFleetInstance.user);
    if(!responseFromFleetInstance.user.api_only){
      throw new Error('The provided API key is invalid');
    }

    let authorization = await ExternalAuthorization.findOrCreate(inputs,inputs);

    this.res.cookie('state', generatedStateForThisRequest, {signed: true});
    this.res.cookie('oauthSourceIdForFleet', inputs.emailAddress, {signed: true});

    let vantaAuthorizationRequestURL = `https://app.vanta.com/oauth/authorize?client_id=${sails.config.custom.vantaAuthorizationClientId}&scope=connectors.self:write-resource connectors.self:read-resource&state=${generatedStateForThisRequest}&source_id=${encodeURIComponent(inputs.emailAddress)}&redirect_uri=${url.resolve(sails.config.custom.baseUrl, '/vanta-callback')}&response_type=code`;


    return vantaAuthorizationRequestURL;
  }


};

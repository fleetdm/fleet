module.exports = {


  friendlyName: 'Get vpp app metadata',


  description: 'Proxies authenticated requests from Fleet instances to the Apple App store API.',


  inputs: {
    storeRegion: {
      type: 'string',
      required: true,
      description: 'The App store region that a proxied request will be sent to.',
    },

    platform: {
      type: 'string',
      description: 'The platform the specified app(s) runs on',
    },

    additionalPlatforms: {
      type: 'string',
      description: 'A comma separated list of platforms that are included in the proxied request.'
    },

    ids: {
      type: 'string',
      description: 'A comma separated list of IDs of app store apps to include in the response.'
    },

    extend: {
      type: {},
      description: 'An object containing the name of additional attributes to include in the API response.'
    },

    vppToken: {
      type: 'string',
      description: 'A VPP token used to authenticate requests to the Apple API on behalf of a Fleet instance.',
      required: true,
    }
  },


  exits: {
    success: {
      description: 'App metadata was sent to the Fleet server',
      outputType: {},
    },
    missingAuthHeader: {
      description: 'This request is missing an authorization header.',
      responseType: 'unauthorized'
    },
    invalidFleetServerSecret: {
      description: 'Invalid authentication token.',
      responseType: 'unauthorized',
    },
    missingVppToken: {
      description: 'This request is missing a VPP app token',
      responseType: 'badRequest',
    },
  },


  fn: async function ({storeRegion, ids, platform, additionalPlatforms, extend, vppToken}) {

    // Validate the provided fleetServerSecret
    let authHeader = this.req.get('authorization');
    let fleetServerSecret;
    if (authHeader && authHeader.startsWith('Bearer')) {
      fleetServerSecret = authHeader.replace('Bearer', '').trim();
    } else {
      throw 'missingAuthHeader';
    }

    let thisFleetInstance = await FleetInstanceUsingVpp.findOne({
      fleetServerSecret: fleetServerSecret
    });

    if(!thisFleetInstance) {
      throw 'invalidFleetServerSecret';
    }


    let nowAt = Date.now();
    let nowAtInSeconds = Math.floor(nowAt / 1000);

    let expiresAtInSeconds = nowAtInSeconds + 60;

    let tokenForThisRequest = require('jsonwebtoken').sign(
      {
        iss: sails.config.custom.vppProxyTokenTeamId,
        exp: expiresAtInSeconds,
        iat: nowAtInSeconds,
      },
      sails.config.custom.vppProxyTokenPrivateKey,
      {
        algorithm: 'ES256',
        keyid: sails.config.custom.vppProxyTokenKeyId,
      }
    );


    let responseFromAppleApi = await sails.helpers.http.get.with({
      url: `https://api.ent.apple.com/v1/catalog/${storeRegion}/stoken-authenticated-apps`,
      data: {
        ids,
        platform,
        additionalPlatforms,
        extend,
      },
      headers: {
        'Authorization': `Bearer ${tokenForThisRequest}`,
        'Cookie': `${vppToken}`,
      }
    })
    .tolerate((err)=>{
      sails.log.warn(`When a Fleet instance sent a proxied request to the Apple App Store API, an error occured. Full error: ${require('util').inspect(err)}`);
      return err;
    });

    return responseFromAppleApi;

  }


};

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


  fn: async function ({storeRegion, ids, platform, additionalPlatforms, extend}) {


    // Validate the fleetServerSecret provided in the authorization header.
    let authHeader = this.req.get('authorization');
    let fleetServerSecret;
    if (authHeader && authHeader.startsWith('Bearer')) {
      fleetServerSecret = authHeader.replace('Bearer', '').trim();
    } else {
      // If no fleetServerSecret was sent, return a missingAuthHeader (unauthorized) response to the Fleet server.
      throw 'missingAuthHeader';
    }

    // Validate the provided fleetServerSecret
    try {
      require('jsonwebtoken').verify(
        fleetServerSecret,
        sails.config.custom.vppProxyAuthenticationPublicKey,
        { algorithm: 'ES256' }
      );
    } catch(unusedErr) {
      // If there is an error parsing the provided fleetServerSecret, return a invalidFleetServerSecret response.
      throw 'invalidFleetServerSecret';
    }

    let vppToken = this.req.get('vpp-token');
    if(!vppToken) {
      // If no vpp-token header was included return a missingVppToken (badRequest) response to the Fleet instance.
      throw 'missingVppToken';
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

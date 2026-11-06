module.exports = {


  friendlyName: 'Authenticate Fleet server',


  description: 'Identify the Fleet server asking the configuration profile generator for a profile, from its Fleet Premium license key or its registration secret.',


  inputs: {

    req: {
      type: 'ref',
      required: true,
      description: 'The request, for its Authorization header.',
    },

    fleetLicenseKey: {
      type: 'string',
      description: 'The license key the Fleet server sent, if any.',
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Fleet server',
      outputDescription: 'Either the `sub` of a Fleet Premium license (`licenseSubject`), or the Fleet Free registration (`registration`) to limit.',
      outputType: 'ref',
    },

    couldNotAuthenticate: {
      description: 'Neither a valid Fleet Premium license key nor a known registration secret was sent.',
    },

  },


  fn: async function ({req, fleetLicenseKey}) {

    let crypto = require('crypto');

    if(fleetLicenseKey) {
      let license;
      try {
        // Tier and expiry come from the verified license, never from anything else Fleet sends.
        license = require('jsonwebtoken').verify(
          fleetLicenseKey,
          sails.config.custom.licenseKeyGeneratorPublicKey,
          { algorithms: ['ES256'], issuer: 'Fleet Device Management Inc.' }
        );
      } catch(err) {
        sails.log.info('The configuration profile generator could not verify a Fleet license:', err.name, err.message);
      }
      // "basic" is the deprecated name for Fleet Premium, and still appears on older licenses.
      if(license && _.contains(['premium', 'trial', 'basic'], license.tier)) {
        return { licenseSubject: license.sub };
      }
    }

    let authHeader = req.get('authorization');
    // Authentication schemes are case-insensitive.
    if(!authHeader || !/^bearer /i.test(authHeader)) {
      throw 'couldNotAuthenticate';
    }
    let fleetServerSecret = authHeader.replace(/^bearer /i, '').trim();
    if(!fleetServerSecret) {
      throw 'couldNotAuthenticate';
    }
    let registration = await ProfileGeneratorRegistration.findOne({
      fleetServerSecretHash: crypto.createHash('sha256').update(fleetServerSecret).digest('hex'),
    });
    if(!registration) {
      throw 'couldNotAuthenticate';
    }
    return { registration };

  }


};

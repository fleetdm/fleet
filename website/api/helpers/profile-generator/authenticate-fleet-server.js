module.exports = {


  friendlyName: 'Authenticate Fleet server',


  description: 'Identify the Fleet server asking the configuration profile generator for a profile, from its Fleet Premium license key or its registration secret.',


  extendedDescription: 'Either credential is counted on a usage row.  A registration\'s row is created when the Fleet server registers, and a license\'s on its first request.',


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
      outputFriendlyName: 'Usage',
      outputDescription: 'The ProfileGeneratorUsage record to count the request on: a Fleet Premium license\'s, or a Fleet Free registration\'s.',
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
        let credentialHash = crypto.createHash('sha256').update(fleetLicenseKey.split('.').slice(0, 2).join('.')).digest('hex');
        let licenseUsage = await ProfileGeneratorUsage.findOne({ credentialType: 'license', credentialHash });
        if(!licenseUsage) {
          // Fleet asks for a profile and for its name and description at the same time, so two first requests
          // for one license arrive together.  The one that loses the race reads the row the other created.
          licenseUsage = await ProfileGeneratorUsage.create({
            credentialType: 'license',
            credentialHash,
            organization: _.isString(license.sub) ? license.sub : '',
          }).fetch()
          .tolerate('E_UNIQUE');
          if(!licenseUsage) {
            licenseUsage = await ProfileGeneratorUsage.findOne({ credentialType: 'license', credentialHash });
          }
        }
        return licenseUsage;
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
    // Only a registration's hash is a hash of a secret.  A license's is a hash of its header and payload, so
    // without the type here, a license key with its signature cut off would work as a bearer token.
    let registrationUsage = await ProfileGeneratorUsage.findOne({
      credentialType: 'registration',
      credentialHash: crypto.createHash('sha256').update(fleetServerSecret).digest('hex'),
    });
    if(!registrationUsage) {
      throw 'couldNotAuthenticate';
    }
    return registrationUsage;

  }


};

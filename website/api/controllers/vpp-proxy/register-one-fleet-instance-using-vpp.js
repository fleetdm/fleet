module.exports = {


  friendlyName: 'Register one Fleet instance for using VPP',


  description: 'Creates a registration for a Fleet instance in the website\'s database and returns a generated secret to authenticate future requests.',



  inputs: {
    fleetServerUrl: {
      type: 'string',
      required: true,
    },
    fleetLicenseKey: {
      type: 'string',
      required: true,
    }
  },


  exits: {
    success: {
      description: 'A Fleet instance\'s VPP proxy registration was successfully submitted.',
      outputType: {
        fleetServerSecret: 'string',
      },
    },
    couldNotVerifyLicense: {
      description: 'The Fleet license key could not be verified.',
      responseType: 'unauthorized',
    },
    invalidFleetServerUrl: {
      description: 'The provided Fleet server URL does not appear to be a URL.',
      responseType: 'badRequest',
    },
  },


  fn: async function ({fleetServerUrl, fleetLicenseKey}) {

    // Validate provided fleetLicenseKey
    try {
      require('jsonwebtoken').verify(
        fleetLicenseKey,
        sails.config.custom.licenseKeyGeneratorPublicKey,
        { algorithm: 'ES256' }
      );
    } catch(unusedErr) {
      // If there is an error parsing the provided fleetLicenseKey, return a couldNotVerifyLicense response.
      throw 'couldNotVerifyLicense';
    }

    // validate Fleet server URL
    try {
      new URL(fleetServerUrl);
    } catch(unusedErr) {
      throw 'invalidFleetServerUrl';
    }


    // Generate a new FleetServerSecret for this Fleet instance.
    let expiresAtInSeconds = Math.floor((Date.now() + (1000 * 60 * 60 * 24 * 365)) / 1000);
    let nowAtInSeconds = Math.floor(Date.now() / 1000);

    let fleetServerSecret = require('jsonwebtoken').sign(
      {
        iss: 'Fleet VPP proxy',
        exp: expiresAtInSeconds,
        iat: nowAtInSeconds,
      },
      {
        key: sails.config.custom.vppProxyAuthenticationPrivateKey,
        passphrase: sails.config.custom.vppProxyAuthenticationPassphrase
      },
      {
        algorithm: 'ES256',
      }
    );

    // Return the generated fleetServerSecret
    return {
      fleetServerSecret,
    };

  }


};

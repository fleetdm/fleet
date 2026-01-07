module.exports = {


  friendlyName: 'Register one Fleet instance for using VPP',


  description: 'Creates a registration for a Fleet instance in the website\'s database and returns a generated secret to authenticate future requests.',



  inputs: {
    fleetServerUrl: {
      type: 'string',
      required: true,
    },
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
  },


  fn: async function ({fleetServerUrl}) {

    // Get the Fleet license key that was sent in the Authorization header as a bearer token.
    let authHeader = this.req.get('authorization');
    let fleetLicenseKey;

    if (authHeader && authHeader.startsWith('Bearer')) {
      fleetLicenseKey = authHeader.replace('Bearer', '').trim();
    } else {
      throw 'couldNotVerifyLicense';
    }

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

    // Generate a new FleetServerSecret for this Fleet instance.
    let fleetServerSecret = sails.helpers.strings.random.with({len: 30});

    // Create a new database record for this Fleet instance.
    await FleetInstanceUsingVpp.create({fleetInstanceUrl: fleetServerUrl, fleetServerSecret});

    // Return the generated fleetServerSecret
    return {
      fleetServerSecret,
    };

  }


};

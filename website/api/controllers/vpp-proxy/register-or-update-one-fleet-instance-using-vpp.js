module.exports = {


  friendlyName: 'Register or update one fleet instance using vpp',


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
      description: 'The provided Fleet license key could not be verified.',
      responseType: 'unauthorized',
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

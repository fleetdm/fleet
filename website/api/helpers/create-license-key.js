module.exports = {


  friendlyName: 'Create license key',


  description: '',


  inputs: {

    numberOfHosts: {
      type: 'number',
      required: true,
    },

    organization: {
      type: 'string',
      required: true,
    },

    expiresAt: {
      type: 'number',
      required: true,
      description: 'A JS timestamp representing when this license will expire.'
    }

  },


  exits: {

    success: {
      outputType: 'string',
    },

  },


  fn: async function ({numberOfHosts, organization, expiresAt}) {

    let jwt = require('jsonwebtoken');

    let expirationTimestampInSeconds = (expiresAt / 1000);
    let token = jwt.sign(
      {
        iss: 'Fleet Device Management Inc.',
        exp: expirationTimestampInSeconds,
        sub: organization,
        devices: numberOfHosts,
        note: 'Created with Fleet License key dispenser',
        tier: 'premium',
      },
      {
        key: sails.config.custom.licenseKeyGeneratorPrivateKey,
        passphrase: sails.config.custom.licenseKeyGeneratorPassphrase
      },
      { algorithm: 'ES256' }
    );


    return token;

  }


};


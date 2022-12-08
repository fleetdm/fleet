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
      description: 'A JS Timestamp representing when this license will expire.'
    }

  },


  exits: {

    success: {
      outputType: 'string',
    },

  },


  fn: async function (inputs) {

    let jwt = require('jsonwebtoken');

    let token = jwt.sign(
      {
        iss: 'Fleet Device Management Inc.',
        exp: inputs.expiresAt,
        sub: inputs.organization,
        devices: inputs.numberOfHosts,
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


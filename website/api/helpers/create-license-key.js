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
    },

    partnerName: {
      type: 'string',
      description: 'The name of the partner who will be reselling this genereated license.',
      extendedDescription: 'This input is only used by the admin license generator tool.',
    }

  },


  exits: {

    success: {
      outputType: 'string',
    },

  },


  fn: async function ({numberOfHosts, organization, expiresAt, partnerName}) {

    let jwt = require('jsonwebtoken');

    let expirationTimestampInSeconds = Math.floor(expiresAt / 1000);
    let token = jwt.sign(
      {
        iss: 'Fleet Device Management Inc.',
        exp: expirationTimestampInSeconds,
        sub: organization,
        devices: numberOfHosts,
        note: 'Created with Fleet License key dispenser',
        tier: 'premium',
        partner: partnerName // If this value is undefined, it will not be included in the generated token.
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


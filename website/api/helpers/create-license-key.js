module.exports = {


  friendlyName: 'Create license key',


  description: '',


  inputs: {

    quoteId: {
      type: 'string',
      required: true,
    },

    validTo: {
      type: 'number',
      required: true,
      description: 'A JS Timestamp representing when this license will expire.'
    }

  },


  exits: {

    success: {
      description: 'All done.',
    },

  },


  fn: async function (inputs) {


    let jwt = require('jsonwebtoken');

    let quoteToCreateLicenseFrom = await Quote.findOne({id: inputs.quoteId});

    let licenseOpts = {
      iss: 'Fleet Device Management Inc.',
      exp: inputs.validTo,
      sub: quoteToCreateLicenseFrom.organization,
      devices: quoteToCreateLicenseFrom.numberOfHosts,
      note: 'Created with Fleet License key dispenser',
      tier: 'premium',
    };
    let token = jwt.sign(
      licenseOpts,
      {key: sails.config.custom.licenseKeyGeneratorPrivateKey, passphrase: sails.config.custom.licenseKeyGeneratorPassphrase },
      { algorithm: 'ES256' },
    );


    return token;

  }


};


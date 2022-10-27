module.exports = {


  friendlyName: 'Generate license key',


  description: 'Generates a Fleet Premium license key',


  inputs: {
    numberOfHosts: {
      type: 'number',
      required: true,
    },

    organization: {
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
      outputType: 'string',
    },
  },


  fn: async function ({numberOfHosts, organization, validTo}) {

    let licenseKey = await sails.helpers.createLicenseKey.with({
      numberOfHosts: numberOfHosts,
      organization: organization,
      validTo: validTo
    });

    return licenseKey;
  }


};

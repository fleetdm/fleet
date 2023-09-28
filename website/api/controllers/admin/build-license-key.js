module.exports = {


  friendlyName: 'Build license key',


  description: 'Build and return a Fleet Premium license key.',


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
      description: 'A JS timestamp representing when this license will expire.',
    },

    partnerName: {
      type: 'string',
      description: 'The name of the partner that will be reselling the generated license.',
    }
  },


  exits: {
    success: {
      outputFriendlyName: 'License key',
      outputType: 'string',
    },
  },


  fn: async function ({numberOfHosts, organization, expiresAt, partnerName}) {

    let licenseKey = await sails.helpers.createLicenseKey.with({
      numberOfHosts: numberOfHosts,
      organization: organization,
      expiresAt: expiresAt,
      partnerName: partnerName,
    });

    return licenseKey;
  }


};

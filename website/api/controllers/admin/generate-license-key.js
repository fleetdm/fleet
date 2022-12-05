module.exports = {


  friendlyName: 'Generate license key',// FUTURE: Rename this to avoid confusion w/ generators.  For example: 'Build license key'


  description: 'Generate and return a Fleet Premium license key.',


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
      description: 'A JS timestamp representing when this license will expire.'
    }
  },


  exits: {
    success: {
      outputFriendlyName: 'License key',
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

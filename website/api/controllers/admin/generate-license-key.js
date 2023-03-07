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

    tier: {
      type: 'string',
      isIn: ['premium', 'trial', 'ultimate'],
      required: true,
    },

    expiresAt: {
      type: 'number',
      required: true,
      description: 'A JS timestamp representing when this license will expire.',
    }
  },


  exits: {
    success: {
      outputFriendlyName: 'License key',
      outputType: 'string',
    },
  },


  fn: async function ({numberOfHosts, organization, expiresAt, tier}) {

    let licenseKey = await sails.helpers.createLicenseKey.with({
      numberOfHosts: numberOfHosts,
      organization: organization,
      tier: tier,
      expiresAt: expiresAt
    });

    return licenseKey;
  }


};

module.exports = {


  friendlyName: 'View generate license',


  description: 'Display "Generate license" page, an admin tool for generating Fleet Premium licenses',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/generate-license'
    },

  },


  fn: async function () {

    // Throw an error if the licenseKeyGeneratorPrivateKey or licenseKeyGeneratorPassphrase are missing.
    if(!sails.config.custom.licenseKeyGeneratorPrivateKey) {
      throw new Error('Missing config variable: The license key generator private key missing (sails.config.custom.licenseKeyGeneratorPrivateKey)! To use this tool, a license key generator private key is required.');
    }

    if(!sails.config.custom.licenseKeyGeneratorPassphrase) {
      throw new Error('Missing config variable: The license key generator passphrase missing(sails.config.custom.licenseKeyGeneratorPassphrase)! To use this tool, a license key generator passphrase is required.');
    }

    // Respond with view.
    return {};
  }


};

module.exports = {


  friendlyName: 'Get signed APNS certificate',


  description: 'Returns a generated zip archive containing a signed APNS certificate.',


  inputs: {

    email: {
      required: true,
      type: 'string',
      description: 'The email address provided with this request',
      isEmail: true,
    },

    orgName: {
      required: true,
      type: 'string',
      description: 'The name of the organization that is sending this request',
      example: 'Fleet Device Management'
    }
  },


  exits: {

    success: {
      description: 'The message was sent successfully.'
    },

    invalidEmailDomain: {
      description: 'The email address provided could not be used for MDM verification.'
    },

  },

  fn: async function({email, orgName}) {
    let path = require('path');

    let binaryFileExists = await sails.helpers.fs.exists(path.resolve(sails.config.appPath, '.tools/mdm-gen-cert'));

    if(!binaryFileExists) {
      throw new Error('Could not generate signed APNS certificate. The mdm-gen-cert binary is missing.'); // TODO: Why would this happen?
    }

    if(!sails.config.custom.mdmVendorCertPem) {
      throw new Error('Cannot generate signed APNS certificate: The vendor certificate PEM (sails.config.custom.mdmVendorCertPem) is missing!')
    }

    if(!sails.config.custom.mdmVendorKeyPem) {
      throw new Error('Cannot generate signed APNS certificate: The vendor key PEM (sails.config.custom.mdmVendorKeyPem) is missing!')
    }

    if(!sails.config.custom.mdmVendorKeyPassphrase) {
      throw new Error('Cannot generate signed APNS certificate: The vendor key passphrase (sails.config.custom.mdmVendorKeyPassphrase) is missing!')
    }



    // Get the domain of the provided email
    let emailDomain = email.split('@')[1];
    // If the email domain is in the list of disallowed email domains list, we'll throw an error
    if(_.includes(sails.config.custom.freeEmailDomains, emailDomain.toLowerCase())){
      return 'invalidEmailDomain';
    }

    let outputFolder = sails.config.appPath+'/.tools/generated/';

    // Make sure the directory for generated zip files exists.
    await sails.helpers.fs.ensureDir(outputFolder);

    let signedAPNSCertOutputPath = outputFolder + await sails.helpers.strings.random.with({len:6})+'.zip'; // TODO

    // Use sails.helpers.process.executeCommand to run the mdm-gen-cert binary.
    await sails.helpers.process.executeCommand.with({
      command: `./.tools/mdm-gen-cert --out ${signedAPNSCertOutputPath} --email ${email}`,
      dir: sails.config.appPath,
      timeout: 10000,
      environmentVars: {
        VENDOR_CERT_PEM: sails.config.custom.mdmVendorCertPem,
        VENDOR_KEY_PEM: sails.config.custom.mdmVendorKeyPem,
        VENDOR_KEY_PASSPHRASE: sails.config.custom.mdmVendorKeyPassphrase,
      },
    });

    // Stream the generated zip file to a variable
    let downloading = await sails.helpers.fs.readStream(signedAPNSCertOutputPath);

    // Set the attachement filename
    this.res.attachment(`Cetificate for ${orgName}.zip`);

    // Respond with the generated zip file, and delete it.
    await sails.helpers.fs.rmrf(signedAPNSCertOutputPath);
    return downloading;
  }

};

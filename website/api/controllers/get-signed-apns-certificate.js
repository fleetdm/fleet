module.exports = {


  friendlyName: 'Get signed APNS certificate',


  description: 'This endpoint will run the ',


  inputs: {

    email: {
      required: true,
      type: 'string',
      description: 'The email address provided with this request',
      isEmail: true,
    },

    org_name: { //eslint-disable-line camelcase
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

  //eslint-disable-next-line camelcase
  fn: async function({email, org_name}) {

    let path = require('path');


    let binaryFileExists = await sails.helpers.fs.exists(path.resolve(sails.config.appPath, '.tools/mdm-gen-cert'));

    if(!binaryFileExists) {
      throw new Error('Could not generate signed APNS certificate. The mdm-gen-cert binary is missing.'); // TODO: Why would this happen?
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
      timeout: 10000, // TODO
      environmentVars: {
        VENDOR_CERT_PEM: sails.config.custom.mdmVendorCert,
        VENDOR_KEY_PEM: sails.config.custom.mdmVendorKey,
        VENDOR_KEY_PASSPHRASE: sails.config.custom.mdmVendorPassphrase,
      },
    });

    // Stream the generated zip file to a variable
    let downloading = await sails.helpers.fs.readStream(signedAPNSCertOutputPath);

    // Set the attachement filename
    this.res.attachment(`Cetificate for ${org_name}.zip`);//eslint-disable-line camelcase

    // Respond with the generated zip file, and delete it.
    await sails.helpers.fs.rmrf(signedAPNSCertOutputPath);
    return downloading;
  }

};

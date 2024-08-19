module.exports = {


  friendlyName: 'Deliver Apple CSR',


  description: 'Generate and optionally deliver a signed certificate signing request to a requesting user\'s email address.',

  extendedDescription: 'Uses the mdm-gen-cert binary to generate a signed CSR for the user and optionally sends the result to the requesting user\'s email address',


  inputs: {
    unsignedCsrData: {
      required: true,
      type: 'string',
      description: 'Base64 encoded CSR submitted from the Fleet server or `fleetctl` on behalf of the user.'
    },
    deliveryMethod: {
      type: 'string',
      description: 'How the signed CSR will be delivered to the user. ',
      defaultsTo: 'email',
      isIn: ['email', 'json']
    }
  },


  exits: {

    success: {
      description: 'Signed the provided CSR.',
      outputType: {csr:'string'},
      outputFriendlyName: 'Signed CSR',
    },

    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },

    badRequest: {
      responseType: 'badRequest'
    }

  },

  fn: async function({unsignedCsrData, deliveryMethod}) {
    let path = require('path');

    let signingToolExists = await sails.helpers.fs.exists(path.resolve(sails.config.appPath, '.tools/mdm-gen-cert'));

    if(!signingToolExists) {
      throw new Error('Could not generate signed CSR: The mdm-gen-cert binary is missing.');
    }

    // Throw an error if we're missing any config variables.
    if(!sails.config.custom.mdmVendorCertPem) {
      throw new Error('Could not generate signed CSR: The vendor certificate PEM (sails.config.custom.mdmVendorCertPem) is missing.');
    }

    if(!sails.config.custom.mdmVendorKeyPem) {
      throw new Error('Could not generate signed CSR: The vendor key PEM (sails.config.custom.mdmVendorKeyPem) is missing.');
    }

    if(!sails.config.custom.mdmVendorKeyPassphrase) {
      throw new Error('Could not generate signed CSR: The vendor key passphrase (sails.config.custom.mdmVendorKeyPassphrase) is missing.');
    }


    // Use sails.helpers.process.executeCommand to run the mdm-gen-cert binary.
    let generateCertificateCommand = await sails.helpers.process.executeCommand.with({
      command: `./.tools/mdm-gen-cert`,
      dir: sails.config.appPath,
      timeout: 10000,
      environmentVars: {
        VENDOR_CERT_PEM: sails.config.custom.mdmVendorCertPem,
        VENDOR_KEY_PEM: sails.config.custom.mdmVendorKeyPem,
        VENDOR_KEY_PASSPHRASE: sails.config.custom.mdmVendorKeyPassphrase,
        CSR_BASE64: unsignedCsrData
      },
    }).intercept((err)=>{
      return new Error(`When trying to generate a signed CSR for a user, an error occured while running the mdm-gen-cert command. Full error: ${err}`);
    });

    // Parse the JSON result from the mdm-gen-cert command
    let generateCertificateResult = JSON.parse(generateCertificateCommand.stdout);
    // Throw an error if the result from the mdm-gen-cert command is missing an email value.
    if(!generateCertificateResult.email) {
      throw new Error('When trying to generate a signed CSR for a user, the result from the mdm-gen-cert command did not contain a email.');
    }
    // Throw an error if the result from the mdm-gen-cert command is missing an org value.
    if(!generateCertificateResult.org) {
      throw new Error('When trying to generate a signed CSR for a user, the result from the mdm-gen-cert command did not contain an organization name');
    }
    // Throw an error if the result from the mdm-gen-cert command is missing an request value.
    if(!generateCertificateResult.request) {
      throw new Error('When trying to generate a signed CSR for a user, the result from the mdm-gen-cert command did not contain a certificate');
    }

    // Check to make sure that the email included in the result is a valid email address.
    try {
      CertificateSigningRequest.validate('emailAddress', generateCertificateResult.email);
    } catch (err) {
      if (err.code === 'E_VIOLATES_RULES') {
        throw 'badRequest';
      } else {
        throw err;
      }
    }

    // Get the domain from the provided email
    let emailDomain = generateCertificateResult.email.split('@')[1];

    // If the email domain is in the list of banned email domains list, we'll return the invalidEmailDomain response to the user.
    if(_.includes(sails.config.custom.bannedEmailDomainsForCSRSigning, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }

    // Create a new CertificateSigningRequest record in the database.
    await CertificateSigningRequest.create({
      emailAddress: generateCertificateResult.email,
      organization: generateCertificateResult.org,
    });

    // respBody contains the raw response body content, it defaults to
    // `undefined` as this was the default behavior of this endpoint before the
    // 'deliveryMethod' parameter was introduced.
    var respBody;
    switch(deliveryMethod) {
      case 'json':
        this.res.type('application/json');
        respBody = JSON.stringify({
          csr: Buffer.from(generateCertificateResult.request).toString('base64'),
        });
        break;
      case 'email':
        // Send an email to the user, with the result from the mdm-gen-cert command attached as a plain text file.
        await sails.helpers.sendTemplateEmail.with({
          to: generateCertificateResult.email,
          subject: 'Your certificate signing request from Fleet',
          from: sails.config.custom.fromEmailAddress,
          fromName: sails.config.custom.fromName,
          template: 'email-signed-csr-for-apns',
          templateData: {},
          attachments: [{
            // When the file is provided as an attachment to the Sails helper, it
            // gets decoded, since we need for the signed CSR to be delivered in
            // base64 format, we doubly encode the contents before sending the
            // email.
            contentBytes: Buffer.from(generateCertificateResult.request).toString('base64'),
            name: 'apple-apns-csr.txt',
            type: 'text/plain',
          }],
        }).intercept((err)=>{
          return new Error(`When trying to send a signed CSR to a user (${generateCertificateResult.email}), an error occured. Full error: ${err}`);
        });
        break;
      default:
        throw 'badRequest';
    }

    // Send a message to Slack.
    await sails.helpers.http.post(sails.config.custom.slackWebhookUrlForMDMSignups, {
      text: `An MDM CSR was generated for ${generateCertificateResult.org} - ${generateCertificateResult.email}`
    });


    return respBody;
  }

};

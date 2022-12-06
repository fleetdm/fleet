module.exports = {


  friendlyName: 'Send certificate signing request email',


  description: 'Returns a generated zip archive containing a signed APNS certificate.',


  inputs: {
    csr: {
      required: true,
      type: 'string',
      description: 'Base64 encoded CSR submitted from the Fleet server or `fleetctl` on behalf of the user.'
    }
  },


  exits: {

    success: {
      description: 'Delivered email to specified email address with certificate signing request attached.'
    },

    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.'
    },

  },

  fn: async function({orgName, csr}) {
    let path = require('path');

    let binaryFileExists = await sails.helpers.fs.exists(path.resolve(sails.config.appPath, '.tools/mdm-gen-cert'));

    if(!binaryFileExists) {
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

    const bannedEmailDomainsForCSRSigning = [
      'gmail.com','yahoo.com','hotmail.com','aol.com','hotmail.co.uk','hotmail.fr','msn.com',
      'yahoo.fr','wanadoo.fr','orange.fr','comcast.net','yahoo.co.uk','yahoo.com.br','yahoo.co.in',
      'live.com','rediffmail.com','free.fr','gmx.de','web.de','yandex.ru','ymail.com','libero.it',
      'outlook.com','uol.com.br','bol.com.br','mail.ru','cox.net','hotmail.it','sbcglobal.net',
      'sfr.fr','live.fr','verizon.net','live.co.uk','googlemail.com','yahoo.es','ig.com.br','live.nl',
      'bigpond.com','terra.com.br','yahoo.it','neuf.fr','yahoo.de','alice.it','rocketmail.com',
      'att.net','laposte.net','facebook.com','bellsouth.net','yahoo.in','hotmail.es','charter.net',
      'yahoo.ca','yahoo.com.au','rambler.ru','hotmail.de','tiscali.it','shaw.ca','yahoo.co.jp',
      'sky.com','earthlink.net','optonline.net','freenet.de','t-online.de','aliceadsl.fr','virgilio.it',
      'home.nl','qq.com','telenet.be','me.com','yahoo.com.ar','tiscali.co.uk','yahoo.com.mx','voila.fr',
      'gmx.net','mail.com','planet.nl','tin.it','live.it','ntlworld.com','arcor.de','yahoo.co.id',
      'frontiernet.net','hetnet.nl','live.com.au','yahoo.com.sg','zonnet.nl','club-internet.fr',
      'juno.com','optusnet.com.au','blueyonder.co.uk','bluewin.ch','skynet.be','sympatico.ca',
      'windstream.net','mac.com','centurytel.net','chello.nl','live.ca','aim.com','bigpond.net.au',
      'icloud.com','protonmail.com','zoho.com','proton.me','pm.me','protonmail.ch','tmmbt.net',
    ];

    // Use sails.helpers.process.executeCommand to run the mdm-gen-cert binary.
    let generateCertificateCommand = await sails.helpers.process.executeCommand.with({
      command: `./.tools/mdm-gen-cert`,
      dir: sails.config.appPath,
      timeout: 10000,
      environmentVars: {
        VENDOR_CERT_PEM: sails.config.custom.mdmVendorCertPem,
        VENDOR_KEY_PEM: sails.config.custom.mdmVendorKeyPem,
        VENDOR_KEY_PASSPHRASE: sails.config.custom.mdmVendorKeyPassphrase,
        CSR_BASE64: csr
      },
    }).catch((err)=>{
      throw new Error(`When trying to generate a signed CSR for a user, an error occured. Full error: ${err}`);
    });

    // Parse the JSON result from the mdm-gen-cert command
    let generateCertificateResult = JSON.parse(generateCertificateCommand.stdout);


    // Throw an error if the result from the mdm-gen-cert command is missing an email.
    if(!generateCertificateResult.email) {
      throw new Error('The result from the mdm-gen-cert command did not contain a email')
    }
    // Throw an error if the result from the mdm-gen-cert command is missing an email or request.
    if(!generateCertificateResult.org) {
      throw new Error('The result from the mdm-gen-cert command did not contain an organization name')
    }
    // Throw an error if the result from the mdm-gen-cert command is missing an email or request.
    if(!generateCertificateResult.result) {
      throw new Error('The result from the mdm-gen-cert command did not contain a certificate')
    }


    // Get the domain of the provided email
    let emailDomain = generateCertificateResult.email.split('@')[1];

    // If the email domain is in the list of disallowed email domains list, we'll return the invalidEmailDomain exit.
    if(_.includes(bannedEmailDomainsForCSRSigning, emailDomain.toLowerCase())){
      return 'invalidEmailDomain';
    };

    // Send an email to the user, with the result from the mdm-gen-cert command attached as a plain text file.
    await sails.helpers.sendTemplateEmail.with({
      to: generateCertificateResult.email,
      subject: 'Your certificate signing request from Fleet',// TODO
      from: sails.config.custom.fromEmailAddress,
      fromName: sails.config.custom.fromName,
      template: 'email-signed-csr-for-apns',
      templateData: {},
      attachments: [{
        contentBytes: generateCertificateResult.result,
        name: 'apple-apns-request.txt',
        type: 'text/plain',
      }],
    }).catch((err)=>{
      throw new Error(`When trying to send a signed CSR to a user (${generateCertificateResult.email}), and error occured. Full error: ${err}`);
    });

    // Create a new CertificateSigningRequest record in the database.
    await CertificateSigningRequest.createOne({emailAddress: generateCertificateResult.email, organization: generateCertificateResult.org});

    // Send a message to the #mdm-csr-signups channel
    await sails.helpers.http.post(sails.config.custom.slackWebhookUrlForMDMSignups, {
      text: `An MDM CSR was generated for ${generateCertificateResult.org} - ${generateCertificateResult.email}`
    });


    return;
  }

};

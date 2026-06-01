module.exports = {


  friendlyName: 'Deliver deal registration submission',


  description: 'Sends an email with the contents of a deal registration form submission',


  inputs: {
    submittersFirstName: {type: 'string', required: true},
    submittersLastName: {type: 'string', required: true},
    submittersEmailAddress: {type: 'string', isEmail: true, required: true},
    submittersOrganization: {type: 'string', required: true},
    submitterIsExistingPartner: {type: 'string', required: true},

    customersOrganization: {type: 'string', required: true},
    customersName: {type: 'string', required: true},
    customersEmailAddress: {type: 'string', isEmail: true, required: true},

    dealStage: {type: 'string', required: true},
    expectedClose: {type: 'string', required: true},
    numberOfHosts: {type: 'string', required: true},

    platforms: {type: {}, required: true},
    useCase: {type: {}, required: true},
    notes: {type: 'string', defaultsTo: 'N/A'},

    websiteUrl: {
      type: 'string',
      description: 'Honeypot field. If filled, the submission is silently discarded.'
    },
  },


  exits: {
    success: {description: 'A deal registration email was successfully sent.'},
    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },
  },


  fn: async function (inputs) {
    if (inputs.websiteUrl) { return; }// Honeypot input provided — return a success response

    let emailDomain = inputs.submittersEmailAddress.split('@')[1];
    if(_.includes(sails.config.custom.bannedEmailDomainsForWebsiteSubmissions, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }
    if(!sails.config.custom.dealRegistrationContactEmailAddress){
      throw new Error('Missing config variable! Please set sails.config.custom.dealRegistrationContactEmailAddress to be the email address of the person who receives deal registration submissions.');
    }
    let emailTemplateData = _.omit(inputs, ['platforms', 'useCase']);


    // Format the submitted platforms and useCase values into strings.
    let platformFriendlyNamesByPlatformValues = {
      apple: 'Apple (macOS, iOS/iPadOS)',
      windows: 'Windows',
      linux: 'Linux',
      other: 'Others (Android, ChromeOS)',
    };
    let formattedPlatformsString = '';
    for(let selectedPlatform of _.keysIn(inputs.platforms)){
      if(inputs.platforms[selectedPlatform] === true){
        formattedPlatformsString += `<br> ${platformFriendlyNamesByPlatformValues[selectedPlatform]}`;
      }
    }
    emailTemplateData.platforms = formattedPlatformsString;

    let useCaseFriendlyNamesBySubmittedValues = {
      mdm: 'Device management',
      security: 'Security / compliance',
    };
    let formattedUseCaseString = '';
    for(let key of _.keysIn(inputs.useCase)){
      if(inputs.useCase[key] === true){
        formattedUseCaseString += `<br> ${useCaseFriendlyNamesBySubmittedValues[key]}`;
      }
    }

    emailTemplateData.useCase = formattedUseCaseString;


    // send the information to the deal registration contact email address.
    await sails.helpers.sendTemplateEmail.with({
      to: sails.config.custom.dealRegistrationContactEmailAddress,
      subject: 'New deal registration form submission',
      template: 'email-deal-registration',
      templateData: emailTemplateData,
    });
    // All done.
    return;

  }


};

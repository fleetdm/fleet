module.exports = {


  friendlyName: 'Deliver partner registration submission',


  description: '',


  inputs: {
    submittersFirstName: { type: 'string', required: true },
    submittersLastName: { type: 'string', required: true },
    submittersEmailAddress: { type: 'string', required: true, isEmail: true },
    submittersOrganization: { type: 'string', required: true },
    partnerType: { type: 'string', required: true, isIn: ['reseller', 'integrations'] },
    partnerWebsite: { type: 'string', required: true },
    partnerCountry: { type: 'string', required: true },
    notes: {type: 'string', required: true },

    servicesOffered: {type: {}},
    numberOfHosts: {type: 'string'},
    servicesCategory: {type: 'string'},

    websiteUrl: {
      type: 'string',
      description: 'Honeypot field. If filled, the submission is silently discarded.'
    },
  },


  exits: {
    success: {description: 'A partner registration email was successfully sent.'},
    missingInput: {description: 'The form submission is missing a required input', responseType: 'badRequest'},
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

    if(inputs.partnerType === 'reseller') {
      if(!inputs.numberOfHosts) {
        throw 'missingInput';
      }
      if(!inputs.servicesOffered){
        throw 'missingInput';
      }
    } else if(inputs.partnerType === 'integrations') {
      if(!inputs.servicesCategory){
        throw 'missingInput';
      }
    }

    let emailTemplateData = _.omit(inputs, 'servicesOffered');
    let partnerTypeFriendlyNameValuesByFormValue = {
      'reseller': 'Resell or manage devices for customers',
      'integrations': 'Build integrations with Fleet'
    };
    emailTemplateData.goal = partnerTypeFriendlyNameValuesByFormValue[inputs.partnerType];

    // Default to sending these to the configured fromEmailAddress
    let toEmail = sails.config.custom.fromEmailAddress;
    if(inputs.partnerType === 'reseller') {

      let servicesFriendlyNamesByFormValues = {
        mdm: 'MDM / endpoint management',
        securityServices: 'Security services (MSSP)',
        itServices: 'IT services / MSP',
        consulting: 'Consulting',
        other: 'Others (Android, ChromeOS)',
      };

      let servicesOfferedAsAFormattedString = '';
      for(let key of _.keysIn(inputs.servicesOffered)){
        if(inputs.servicesOffered[key] === true){
          servicesOfferedAsAFormattedString += `<br> ${servicesFriendlyNamesByFormValues[key]}`;
        }
      }
      emailTemplateData.servicesOffered = servicesOfferedAsAFormattedString;
      // For new resellers registering, send the information to the deal registration contact email address.
      toEmail = sails.config.custom.dealRegistrationContactEmailAddress;
    }

    await sails.helpers.sendTemplateEmail.with({
      to: toEmail,
      subject: 'New parter registration form submission',
      template: 'email-partner-registration',
      templateData: emailTemplateData,
    });



    // All done.
    return;

  }


};

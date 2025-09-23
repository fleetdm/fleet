module.exports = {


  friendlyName: 'Deliver talk to us form submission',


  description: '',


  inputs: {
    emailAddress: {
      required: true,
      type: 'string',
      description: 'A return email address where we can respond.',
      example: 'hermione@hogwarts.edu'
    },

    firstName: {
      required: true,
      type: 'string',
      description: 'The first name of the human sending this message.',
      example: 'Emma'
    },

    lastName: {
      required: true,
      type: 'string',
      description: 'The last name of the human sending this message.',
      example: 'Watson'
    },

    organization: {
      type: 'string',
      required: true,
      description: 'The organization of the user who submitted the "talk to us" form'
    },

    numberOfHosts: {
      type: 'string',
      required: true,
      description: 'The organization of the user who submitted the "talk to us" form'
    },

    primaryBuyingSituation: {
      type: 'string',
      required: true,
      description: 'What this user will be using Fleet for',
      isIn: [
        'it-major-mdm',
        'it-gap-filler-mdm',
        'it-misc',
        'security-misc',
        'security-vm',
      ],
    },

  },


  exits: {

    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },
    success: {
      decription: 'A user successfully submitted the "Talk to us" form.',
      outputType: 'string',
    }

  },


  fn: async function ({emailAddress, firstName, lastName, organization, numberOfHosts, primaryBuyingSituation}) {

    let emailDomain = emailAddress.split('@')[1];
    if(_.includes(sails.config.custom.bannedEmailDomainsForWebsiteSubmissions, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }
    // Use the getEnriched helper to see if we can find out how many employees this user's company has.
    let enrichmentInformation = await sails.helpers.iq.getEnriched.with({
      emailAddress,
      firstName,
      lastName,
      organization,
    });

    let contactInformation = {
      emailAddress: emailAddress,
      firstName: firstName,
      lastName: lastName,
      organization: organization,
      primaryBuyingSituation: primaryBuyingSituation === 'security-misc' ? 'Endpoint operations - Security' : primaryBuyingSituation === 'it-misc' ? 'Endpoint operations - IT' : primaryBuyingSituation === 'it-major-mdm' ? 'Device management (MDM)' : primaryBuyingSituation === 'it-gap-filler-mdm' ? 'IT - Gap-filler MDM' : primaryBuyingSituation === 'security-vm' ? 'Vulnerability management' : undefined,
      contactSource: 'Website - Contact forms',
      psychologicalStage: '4 - Has use case',
      psychologicalStageChangeReason: 'Website - Contact forms'
    };

    // If we got a employer.numberOfEmployees value from the getEnriched helper, use that to determine which Calendly event to send them to (if it is more than the provided numberOfHosts).
    if(enrichmentInformation.employer && enrichmentInformation.employer.numberOfEmployees && enrichmentInformation.employer.numberOfEmployees > numberOfHosts) {
      if(enrichmentInformation.employer.numberOfEmployees >= 700){
        contactInformation.description = `Submitted the "Talk to us" form and was taken to the Calendly page for the "Talk to us" event because of the number of employees (${enrichmentInformation.employer.numberOfEmployees}) returned by Coresignal. Number of hosts: ${numberOfHosts}`;
        sails.helpers.salesforce.updateOrCreateContactAndAccount.with(contactInformation).exec((err)=>{
          if(err) {
            sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}.`, err);
          }
        });
        return `https://calendly.com/fleetdm/talk-to-us?email=${encodeURIComponent(emailAddress)}&name=${encodeURIComponent(firstName+' '+lastName)}`;
      } else {
        contactInformation.description = `Submitted the "Talk to us" form and was taken to the Calendly page for the "Let\'s get you set up!" event. Number of hosts: ${numberOfHosts}`;
        sails.helpers.salesforce.updateOrCreateContactAndAccount.with(contactInformation).exec((err)=>{
          if(err) {
            sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}.`, err);
          }
        });
        return `https://calendly.com/fleetdm/chat?email=${encodeURIComponent(emailAddress)}&name=${encodeURIComponent(firstName+' '+lastName)}`;
      }
    } else {// If the enrichment helper didn't return a  employer.numberOfEmployees value, use the submitted number of hosts.
      if(numberOfHosts >= 700){
        contactInformation.description = `Submitted the "Talk to us" form and was taken to the Calendly page for the "Talk to us" event. Number of hosts: ${numberOfHosts}`;
        sails.helpers.salesforce.updateOrCreateContactAndAccount.with(contactInformation).exec((err)=>{
          if(err) {
            sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}.`, err);
          }
        });
        return `https://calendly.com/fleetdm/talk-to-us?email=${encodeURIComponent(emailAddress)}&name=${encodeURIComponent(firstName+' '+lastName)}`;
      } else {
        contactInformation.description = `Submitted the "Talk to us" form and was taken to the Calendly page for the "Let\'s get you set up!" event. Number of hosts: ${numberOfHosts}`;
        sails.helpers.salesforce.updateOrCreateContactAndAccount.with(contactInformation).exec((err)=>{
          if(err) {
            sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}.`, err);
          }
        });
        return `https://calendly.com/fleetdm/chat?email=${encodeURIComponent(emailAddress)}&name=${encodeURIComponent(firstName+' '+lastName)}`;
        // FUTURE: create POV here
      }
    }
  }


};

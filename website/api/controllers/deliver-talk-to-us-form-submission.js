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
        'vm',
        'mdm',
        'eo-it',
        'eo-security',
      ],
    },

  },


  exits: {

    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and was not delivered.',
      responseType: 'badRequest'
    },

  },


  fn: async function ({emailAddress, firstName, lastName, organization, numberOfHosts, primaryBuyingSituation}) {


    let emailDomain = emailAddress.split('@')[1];
    if(_.includes(sails.config.custom.bannedEmailDomainsForWebsiteSubmissions, emailDomain.toLowerCase())){
      throw 'invalidEmailDomain';
    }

    // Use timers.setImmediate() to update/create CRM records in the background.
    require('timers').setImmediate(async ()=>{
      await sails.helpers.salesforce.updateOrCreateContactAndAccountAndCreateLead.with({
        emailAddress: emailAddress,
        firstName: firstName,
        lastName: lastName,
        organization: organization,
        numberOfHosts: numberOfHosts,
        primaryBuyingSituation: primaryBuyingSituation === 'eo-security' ? 'Endpoint operations - Security' : primaryBuyingSituation === 'eo-it' ? 'Endpoint operations - IT' : primaryBuyingSituation === 'mdm' ? 'Device management (MDM)' : primaryBuyingSituation === 'vm' ? 'Vulnerability management' : undefined,
        leadSource: 'Website - Contact forms',
        leadDescription: `Submitted the "Talk to us" form.`,
      }).tolerate((err)=>{
        sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}. Error:`, err.raw);
      });
    });//_∏_  (Meanwhile...)

    return;
  }


};

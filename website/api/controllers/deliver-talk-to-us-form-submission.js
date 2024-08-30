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
    // Set a default psychological stage and change reason.
    let psyStageAndChangeReason = {
      psychologicalStage: '4 - Has use case',
      psychologicalStageChangeReason: 'Website - Contact forms'
    };
    if(this.req.me){
      // If this user is logged in, check their current psychological stage, and if it is higher than 4, we won't set a psystage.
      // This way, if a user has a psytage >4, we won't regress their psystage because they submitted this form.
      if(['4 - Has use case', '5 - Personally confident', '6 - Has team buy-in'].includes(this.req.me.psychologicalStage)) {
        psyStageAndChangeReason = {};
      }
    }
    if(numberOfHosts >= 700){
      sails.helpers.salesforce.updateOrCreateContactAndAccountAndCreateLead.with({
        emailAddress: emailAddress,
        firstName: firstName,
        lastName: lastName,
        organization: organization,
        numberOfHosts: numberOfHosts,
        primaryBuyingSituation: primaryBuyingSituation === 'eo-security' ? 'Endpoint operations - Security' : primaryBuyingSituation === 'eo-it' ? 'Endpoint operations - IT' : primaryBuyingSituation === 'mdm' ? 'Device management (MDM)' : primaryBuyingSituation === 'vm' ? 'Vulnerability management' : undefined,
        contactSource: 'Website - Contact forms',
        leadDescription: `Submitted the "Talk to us" form and was taken to the Calendly page for the "Talk to us" event.`,
        ...psyStageAndChangeReason// Only (potentially) set psystage and change reason for >700 hosts.
      }).exec((err)=>{
        if(err) {
          sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}.`, err);
        }
      });
    } else {
      sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
        emailAddress: emailAddress,
        firstName: firstName,
        lastName: lastName,
        organization: organization,
        primaryBuyingSituation: primaryBuyingSituation === 'eo-security' ? 'Endpoint operations - Security' : primaryBuyingSituation === 'eo-it' ? 'Endpoint operations - IT' : primaryBuyingSituation === 'mdm' ? 'Device management (MDM)' : primaryBuyingSituation === 'vm' ? 'Vulnerability management' : undefined,
        contactSource: 'Website - Contact forms',
        description: `Submitted the "Talk to us" form and was taken to the Calendly page for the "Let\'s get you set up!" event.`,
      }).exec((err)=>{
        if(err) {
          sails.log.warn(`Background task failed: When a user submitted the "Talk to us" form, a lead/contact could not be updated in the CRM for this email address: ${emailAddress}.`, err);
        }
      });
    }

    return;
  }


};

module.exports = {


  friendlyName: 'Update or create contact and account and create lead',


  description: 'Updates or creates a contact and account in Salesforce, then uses the IDs of the created records to create a Lead record.',

  extendedDescription: 'This is a wrapper for the update-or-create-contact-and-account and create-lead helpers used to run both of them in the background with timers.setImmediate().',

  inputs: {

    // Find by…
    emailAddress: { type: 'string' },
    linkedinUrl: { type: 'string' },

    // Set…
    firstName: { type: 'string', required: true },
    lastName: { type: 'string', required: true },
    organization: { type: 'string' },
    primaryBuyingSituation: { type: 'string' },
    psychologicalStage: {
      type: 'string',
      isIn: [
        '1 - Unaware',
        '2 - Aware',
        '3 - Intrigued',
        '4 - Has use case',
        '5 - Personally confident',
        '6 - Has team buy-in'
      ]
    },
    psychologicalStageChangeReason: {
      type: 'string',
      example: 'Website - Organic start flow'
    },
    // For new leads.
    leadDescription: {
      type: 'string',
      description: 'A description of what this lead is about; e.g. a contact form message, or the size of t-shirt being requested.'
    },
    contactSource: {
      type: 'string',
      required: true,
      isIn: [
        'Attended a call with Fleet',
        'Event',
        'GitHub - Stared fleetdm/fleet',
        'GitHub - Forked fleetdm/fleet',
        'GitHub - Contributed to fleetdm/fleet',
        'LinkedIn - Reaction',
        'LinkedIn - Comment',
        'LinkedIn - Share',
        'LinkedIn - Liked the LinkedIn company page',
        'Prospecting - AE',
        'Prospecting - Specialist',
        'Prospecting - Meeting service',
        'Website - Chat',
        'Website - Contact forms',
        'Website - Contact forms - Demo - ICP',
        'Website - Contact forms - Demo',
        'Website - GitOps',
        'Website - Newsletter',
        'Website - Sign up',
        'Website - Swag request',
      ],
    },
    numberOfHosts: { type: 'number' },
  },

  exits: {

    success: {
      extendedDescription: 'Note that this deliberately has no return value.',
    },

  },



  fn: async function ({emailAddress, linkedinUrl, firstName, lastName, organization, primaryBuyingSituation, psychologicalStage, psychologicalStageChangeReason, contactSource, leadDescription, numberOfHosts}) {
    if(sails.config.environment !== 'production') {
      sails.log('Skipping Salesforce integration...');
      return;
    }

    let recordIds = await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
      emailAddress,
      firstName,
      lastName,
      organization,
      linkedinUrl,
      primaryBuyingSituation,
      psychologicalStage,
      psychologicalStageChangeReason,
      contactSource,
      description: leadDescription,
    });

    await sails.helpers.salesforce.createLead.with({
      salesforceContactId: recordIds.salesforceContactId,
      salesforceAccountId: recordIds.salesforceAccountId,
      leadDescription,
      leadSource: contactSource,
      numberOfHosts,
    });

    return;
  }


};


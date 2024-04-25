module.exports = {


  friendlyName: 'Create lead',// FUTURE: Retire this in favor of createTask()


  description: 'Create a Lead record in Salesforce representing some kind of action Fleet needs to take for someone, whether based on a signal from their behavior or their explicit request.',


  inputs: {

    salesforceAccountId: { type: 'string', required: true },
    salesforceContactId: { type: 'string', required: true },
    leadDescription: { type: 'string', description: 'A description of what this lead is about; e.g. a contact form message, or the size of t-shirt being requested.' },
    leadSource: { type: 'string', required: true, isIn: ['Website - Contact forms'], },// TODO verify and complete enum


    // FUTURE: Move these off eventually:
    firstName: { type: 'string', required: true, description: 'The first name of the referenced contact.' },
    lastName: { type: 'string', required: true, description: 'The last name of the referenced contact.' },
    emailAddress: { type: 'string', description: 'The email address of the referenced contact.', extendedDescription: 'Included here so that the little Salesforce thingie that shows email and calendar activity shows maximum contact in both the Contact and Lead views.' },
    primaryBuyingSituation: { type: 'string' },
    numberOfHosts: { type: 'number' },

  },


  exits: {

    success: {
      extendedDescription: 'Note that this deliberately has no return value.',
    },

  },


  fn: async function ({salesforceAccountId, salesforceContactId, leadDescription, leadSource, firstName, lastName, emailAddress, primaryBuyingSituation, numberOfHosts}) {
    require('assert')(sails.config.custom.salesforceSecret);// todo

    // TODO
  }


};


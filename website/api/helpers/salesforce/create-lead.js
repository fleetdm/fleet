module.exports = {


  friendlyName: 'Create lead',// FUTURE: Retire this in favor of createTask()


  description: 'Create a Lead record in Salesforce representing some kind of action Fleet needs to take for someone, whether based on a signal from their behavior or their explicit request.',


  inputs: {

    salesforceAccountId: { type: 'string', required: true },
    salesforceContactId: { type: 'string', required: true },
    leadDescription: { type: 'string', description: 'A description of what this lead is about; e.g. a contact form message, or the size of t-shirt being requested.' },
    leadSource: { type: 'string', required: true, isIn: ['Website - Contact forms', 'Website - Sign up', 'Website - Waitlist', 'Website - swag request'], },// TODO verify and complete enum


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
    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    require('assert')(sails.config.custom.salesforceIntegrationPasskey);
    let jsforce = require('jsforce');
    console.log(firstName, lastName, emailAddress, primaryBuyingSituation, numberOfHosts);
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);
    // Get the contact record
    let contactRecord = await salesforceConnection.sobject('Contact')
    .retrieve(salesforceContactId);
    // Verify that the account ID provided is valid.
    let accountRecord = await salesforceConnection.sobject('Account')
    .retrieve(salesforceAccountId);

    // TODO better error messages
    if(contactRecord === null) {
      throw new Error(`When attempting to create a Salesforce lead using the ID of a Contact record, no Contact matching the id provided (${salesforceContactId} was found.`);
    }
    if(accountRecord === null) {
      throw new Error(`When attempting to create a Salesforce lead, no account matching the id provided (${salesforceContactId} could be found`);
    }

    // TODO: wrap this in a try-catch block to handle errors from Salesforce.
    // Create the new Lead record.
    let lead = await salesforceConnection.sobject('Lead')
    .create({
      FirstName: contactRecord.FirstName,
      LastName: contactRecord.LastName,
      Email: contactRecord.Email,
      Website: contactRecord.Website,
      // eslint-disable-next-line camelcase
      of_hosts__c: contactRecord.of_hosts__c,
      // eslint-disable-next-line camelcase
      Primary_buying_scenario__c: contactRecord.Primary_buying_situation__c,
      // eslint-disable-next-line camelcase
      LinkedIn_profile__c: contactRecord.LinkedIn_profile__c,
      Description: leadDescription,
      LeadSource: leadSource,
      // eslint-disable-next-line camelcase
      Contact_associated_by_website__c: salesforceContactId,
      // eslint-disable-next-line camelcase
      Account__c: salesforceAccountId,
      OwnerId: accountRecord.OwnerId
    });
    console.log(`Created lead! ${lead}`);

    // TODO handle duplicate leads:
  }


};


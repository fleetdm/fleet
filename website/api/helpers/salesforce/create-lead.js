module.exports = {


  friendlyName: 'Create lead',// FUTURE: Retire this in favor of createTask()


  description: 'Create a Lead record in Salesforce representing some kind of action Fleet needs to take for someone, whether based on a signal from their behavior or their explicit request.',


  inputs: {

    salesforceAccountId: {
      type: 'string',
      required: true,
      description: 'The ID of the Account record that was found or updated by the updateOrCreateContactAndAccount helper.'
    },
    salesforceContactId: {
      type: 'string',
      required: true
    },
    leadDescription: {
      type: 'string',
      description: 'A description of what this lead is about; e.g. a contact form message, or the size of t-shirt being requested.'
    },
    leadSource: {
      type: 'string',
      required: true,
      isIn: [
        'Website - Contact forms',
        'Website - Sign up',
        'Website - Waitlist',
        'Website - swag request',
      ],
    },
    primaryBuyingSituation: { type: 'string', isin: ['eo-it', 'eo-security', 'mdm', 'vm'] },
    numberOfHosts: { type: 'number' },
  },


  exits: {

    success: {
      extendedDescription: 'Note that this deliberately has no return value.',
    },

  },


  fn: async function ({salesforceAccountId, salesforceContactId, leadDescription, leadSource, primaryBuyingSituation, numberOfHosts}) {
    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    require('assert')(sails.config.custom.salesforceIntegrationPasskey);
    let jsforce = require('jsforce');

    // login to Salesforce
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

    // Get the Contact record.
    let contactRecord = await sails.helpers.flow.build(async ()=>{
      return await salesforceConnection.sobject('Contact')
      .retrieve(salesforceContactId);
    }).intercept((err)=>{
      return new Error(`When attempting to create a new Lead record using an existing Contact record (ID: ${salesforceContactId}), an error occurred when retreiving the specified record. Error: ${err}`);
    });

    // Get the Account record.
    let accountRecord = await sails.helpers.flow.build(async ()=>{
      return await salesforceConnection.sobject('Account')
      .retrieve(salesforceAccountId);
    }).intercept((err)=>{
      return new Error(`When attempting to create a Lead record using an exisitng Account record (ID: ${salesforceAccountId}), An error occured when retreiving the specified record. Error: ${err}`);
    });
    let salesforceAccountOwnerId = accountRecord.OwnerId;

    let primaryBuyingSituationValuesByCodename = {
      'vm': 'Vulnerability management',
      'mdm': 'Device management (MDM)',
      'eo-it': 'Endpoint operations - IT',
      'eo-security': 'Endpoint operations - Security',
    };

    // If numberOfHosts or primaryBuyingSituationToSet was provided, set that value on the new Lead, otherwise fallback to the value on the contact record. (If it has one)
    // Note: If these were not provided and a retreived contact record does not have this information, these values will be set to 'null' and are safe to pass into the sobject('Lead').create method below.
    let numberOfHostsToSet = numberOfHosts ? numberOfHosts : contactRecord.of_hosts__c;
    let primaryBuyingSituationToSet = primaryBuyingSituation ? primaryBuyingSituationValuesByCodename[primaryBuyingSituation] : contactRecord.Primary_buying_situation__c;

    // Create the new Lead record.
    await sails.helpers.flow.build(async ()=>{
      return await salesforceConnection.sobject('Lead')
      .create({
        // Information from inputs:
        Description: leadDescription,
        LeadSource: leadSource,
        Account__c: salesforceAccountId,// eslint-disable-line camelcase
        Contact_associated_by_website__c: salesforceContactId,// eslint-disable-line camelcase
        // Information from contact record:
        FirstName: contactRecord.FirstName,
        LastName: contactRecord.LastName,
        of_hosts__c: numberOfHostsToSet,// eslint-disable-line camelcase
        Primary_buying_scenario__c: primaryBuyingSituationToSet,// eslint-disable-line camelcase
        LinkedIn_profile__c: contactRecord.LinkedIn_profile__c,// eslint-disable-line camelcase
        // Information from the account record:
        OwnerId: salesforceAccountOwnerId
      });
    }).intercept((err)=>{
      return new Error(`Could not create new Lead record. Error: ${err}`);
    });
  }


};


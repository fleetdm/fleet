module.exports = {


  friendlyName: 'Update or create contact and account and create website activity',


  description: 'Updates or creates a contact and account in Salesforce, then creates a Fleet website page view record.',


  inputs: {

    // Find by…
    emailAddress: { type: 'string' },

    // Set…
    firstName: { type: 'string', required: true },
    lastName: { type: 'string', required: true },
    organization: { type: 'string' },

    // For website activity
    pageUrlVisited: { type: 'string', required: true},
  },

  exits: {

    success: {
      extendedDescription: 'Note that this deliberately has no return value.',
    },

  },



  fn: async function ({emailAddress, firstName, lastName, organization, pageUrlVisited}) {
    if(sails.config.environment !== 'production') {
      sails.log.verbose('Skipping Salesforce integration...');
      return;
    }
    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    require('assert')(sails.config.custom.salesforceIntegrationPasskey);

    let recordIds = await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
      emailAddress,
      firstName,
      lastName,
      organization,
    });
    let jsforce = require('jsforce');
    // login to Salesforce
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

    let today = new Date();
    let nowOn = today.toISOString().replace('Z', '+0000');
    // Create the new Fleet website page view record.
    await sails.helpers.flow.build(async ()=>{
      return await salesforceConnection.sobject('fleet_website_page_views__c')
      .create({
        Contact__c: recordIds.salesforceContactId,// eslint-disable-line camelcase
        Page_URL__c: pageUrlVisited,// eslint-disable-line camelcase
        Visited_on__c: nowOn,// eslint-disable-line camelcase
      });
    }).intercept((err)=>{
      return new Error(`Could not create new Fleet website page view record. Error: ${err}`);
    });

    return;
  }


};


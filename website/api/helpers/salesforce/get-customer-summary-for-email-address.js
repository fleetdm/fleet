module.exports = {


  friendlyName: 'Get customer summary for email address',


  description: 'Look up an email address in Salesforce and summarize whether its account is (or was) a Fleet customer.',


  extendedDescription: 'Used to help Fleet team members verify eligibility claims on fleetdm.com/hire. This is a hint for a human, not proof: fleetdm.com does not require email verification.',


  inputs: {

    emailAddress: { type: 'string', required: true, isEmail: true },

  },


  exits: {

    success: {
      outputFriendlyName: 'Summary',
      outputType: 'string',
      outputExample: 'Contact found at "Acme Inc." (Account type: Customer). 2 closed-won opportunities.'
    },

  },


  fn: async function ({emailAddress}) {

    if(sails.config.environment !== 'production') {
      sails.log.verbose('Skipping Salesforce integration...');
      return 'Salesforce lookup skipped (not running in production).';
    }

    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    require('assert')(sails.config.custom.salesforceIntegrationPasskey);

    let jsforce = require('jsforce');
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

    let contactRecord = await salesforceConnection.sobject('Contact').findOne({Email: emailAddress}, ['Id', 'AccountId']);
    if(!contactRecord) {
      contactRecord = await salesforceConnection.sobject('Contact').findOne({
        Last_email_associated_by_fleetdm_com__c: emailAddress,// eslint-disable-line camelcase
      }, ['Id', 'AccountId']);
    }
    if(!contactRecord || !contactRecord.AccountId) {
      return `No Salesforce contact found for ${emailAddress}.`;
    }

    let accountRecord = await salesforceConnection.sobject('Account').findOne({Id: contactRecord.AccountId}, ['Id', 'Name', 'Type']);
    let numClosedWonOpportunities = await salesforceConnection.sobject('Opportunity').count({AccountId: contactRecord.AccountId, IsWon: true});

    return `Contact found at "${accountRecord.Name}" (Account type: ${accountRecord.Type || 'none'}). ${numClosedWonOpportunities} closed-won ${numClosedWonOpportunities === 1 ? 'opportunity' : 'opportunities'}.`;

  }


};

module.exports = {


  friendlyName: 'Receive from Clay',


  description: 'Receive webhook requests from Clay.',


  inputs: {
    webhookSecret: {
      type: 'string',
      required: true,
    },

    // For finding/creating contacts.
    firstName: {
      type: 'string',
      required: true,
    },
    lastName: {
      type: 'string',
      required: true,
    },
    linkedinUrl: {
      type: 'string',
      required: true,
    },
    contactSource: {
      type: 'string',
      required: true
    },
    jobTitle: {
      type: 'string',
    },

    // For creating historical event.
    intentSignal: {
      type: 'string',
      required: true,
    },
    historicalContent: {
      type: 'string',
      required: true,
    },
    historicalContentUrl: {
      type: 'string',
    }
  },


  exits: {
    success: { description: 'Information about LinkedIn activity has successfully been received.' },
  },


  fn: async function ({webhookSecret, firstName, lastName, linkedinUrl, contactSource, jobTitle, intentSignal, historicalContent, historicalContentUrl}) {


    if (!sails.config.custom.clayWebhookSecret) {
      throw new Error('No webhook secret configured!  (Please set `sails.config.custom.zapierWebhookSecret`.)');
    }

    if(webhookSecret !== sails.config.custom.clayWebhookSecret){
      throw new Error('Received unexpected webhook request with webhookSecret set to: '+webhookSecret);
    }


    let recordIds = await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
      firstName,
      lastName,
      linkedinUrl,
      contactSource,
      jobTitle,
    }).intercept((err)=>{
      return new Error(`When the receive-from-clay webhook received information about LinkedIn activity, a contact/account could not be created or updated. Full error: ${require('util').inspect(err)}`);
    });


    let trimmedLinkedinUrl = linkedinUrl.replace(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS, '');

    // Create the new Fleet website page view record.
    let newHistoricalRecord = await sails.helpers.flow.build(async ()=>{

      let jsforce = require('jsforce');

      // login to Salesforce
      let salesforceConnection = new jsforce.Connection({
        loginUrl : 'https://fleetdm.my.salesforce.com'
      });
      await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

      return await salesforceConnection.sobject('fleet_website_page_views__c')
      .create({
        Contact__c: recordIds.salesforceContactId,// eslint-disable-line camelcase
        Account__c: recordIds.salesforceAccountId,// eslint-disable-line camelcase
        Event_type__c: 'Intent signal',// eslint-disable-line camelcase
        Intent_signal__c: intentSignal,// eslint-disable-line camelcase
        Content__c: historicalContent,// eslint-disable-line camelcase
        Content_url__c: historicalContentUrl,// eslint-disable-line camelcase
        Interactor_profile_url__c: trimmedLinkedinUrl,// eslint-disable-line camelcase
      });
    }).intercept((err)=>{
      return new Error(`When the receive-from-clay webhook received information about linkedIn activity, a historical event record could not be created. Full error: ${require('util').inspect(err)}`);
    });

    // All done.
    return {
      historicalRecordId: newHistoricalRecord.id,
      contactId: recordIds.salesforceContactId,
      accountId: recordIds.salesforceAccountId
    };

  }


};

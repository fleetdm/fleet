module.exports = {


  friendlyName: 'Create Historical event',


  description: 'Create a historical event related to a particular contact and account in Salesforce.',


  inputs: {
    // Required values on new historical event records.
    salesforceAccountId: {
      type: 'string',
      required: true,
      extendedDescription: 'This ID of the account associated with this historical event'
    },
    salesforceContactId: {
      type: 'string',
      required: true,
      extendedDescription: 'This ID of the contact associated with this historical event'
    },
    eventType: {
      type: 'string',
      required: true,
      isIn: [
        'Intent signal',
        'Website page view',
      ],
    },

    // For "Website page view" type historical events:
    fleetWebsitePageUrl: {
      type: 'string',
      description: 'The url of the page this user viewed.',
      extendedDescription: 'This input is required when the event type is "Website page view".'
    },
    websiteVisitReason: {
      type: 'string',
      description: ''
    },

    // For "Intent signal" type historical events:
    intentSignal: {
      type: 'string',
      isIn: [
        'Followed the Fleet LinkedIn company page',
        'LinkedIn comment',
        'LinkedIn share',
        'LinkedIn reaction',
        'Fleet channel member in MacAdmins Slack',
        'Fleet channel member in osquery Slack',
        'Implemented a trial key',
        'Engaged with fleetie at community event',
        'Attended a Fleet happy hour',
        'Stared the fleetdm/fleet repo on GitHub',
        'Forked the fleetdm/fleet repo on GitHub',
        'Subscribed to the Fleet newsletter',
        'Attended a Fleet training course'
      ]
    },
    eventContent: {
      type: 'string',
    },
    eventContentUrl: {
      type: 'string',
    },
    linkedinUrl: {
      type: 'string',
    },
  },


  exits: {

    success: {
      outputType: {
        salesforceHistoricalEventId: 'string',
      },
    },

  },


  fn: async function ({ salesforceAccountId, salesforceContactId, eventType, linkedinUrl, intentSignal, eventContent, eventContentUrl, fleetWebsitePageUrl, websiteVisitReason}) {
    // Return undefined if we're not running in a production environment.
    if(sails.config.environment !== 'production') {
      sails.log.verbose('Skipping Salesforce integration...');
      return {
        salesforceHistoricalEventId: undefined
      };
    }

    require('assert')(sails.config.custom.salesforceIntegrationPasskey);
    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    require('assert')(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS);

    // Normalize the provided linkedin url.
    if(linkedinUrl){
      linkedinUrl = linkedinUrl.replace(sails.config.custom.RX_PROTOCOL_AND_COMMON_SUBDOMAINS, '');
    }
    // Check for required values depending on the eventType value.
    if(eventType === 'Intent signal') {
      if(!intentSignal) {
        throw new Error(`A intentSignal value is required when creating "Intent signal" type historical events`);
      }
    } else if(eventType === 'Website page view') {
      if(!fleetWebsitePageUrl){
        throw new Error(`A fleetWebsitePageUrl value is required when creating "Website page view" type historical events`);
      }
    }

    // Use sails.helpers.flow.build to login to salesforce and create the new histrocial event record.
    let newHistoricalRecord = await sails.helpers.flow.build(async ()=>{


      // login to Salesforce
      let jsforce = require('jsforce');
      let salesforceConnection = new jsforce.Connection({
        loginUrl : 'https://fleetdm.my.salesforce.com'
      });
      await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

      return await salesforceConnection.sobject('fleet_website_page_views__c')
      .create({
        Contact__c: salesforceContactId,// eslint-disable-line camelcase
        Account__c: salesforceAccountId,// eslint-disable-line camelcase
        Event_type__c: eventType,// eslint-disable-line camelcase

        Intent_signal__c: intentSignal,// eslint-disable-line camelcase
        Content__c: eventContent,// eslint-disable-line camelcase
        Content_url__c: eventContentUrl,// eslint-disable-line camelcase
        Interactor_profile_url__c: linkedinUrl,// eslint-disable-line camelcase

        Page_URL__c: fleetWebsitePageUrl,// eslint-disable-line camelcase
        Website_visit_reason__c: websiteVisitReason// eslint-disable-line camelcase
      });
    }).intercept((err)=>{
      return new Error(`An error occured when creating a new Historical event record in Salesforce. full error ${require('util').inpsect(err, {depth: null})}`);
    });



    return newHistoricalRecord.id;

  }


};


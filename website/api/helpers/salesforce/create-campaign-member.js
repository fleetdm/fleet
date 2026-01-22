module.exports = {


  friendlyName: 'Create campaign member',


  description: 'Creates a "Campaign Member" in Salesforce',


  inputs: {
    salesforceAccountId: {
      type: 'string',
      required: true,
      extendedDescription: 'This ID of the account associated with the contact that will be added to a campaign'
    },
    salesforceContactId: {
      type: 'string',
      required: true,
      extendedDescription: 'This ID of the contact that will be added to a campaign'
    },
    campaignName: {
      type: 'string',
      required: true,
    },
  },


  exits: {

    success: {
      description: 'All done.',
    },

    campaignNotFound: {
      description: 'The provided campaign name did not match any active campaigns records in Salesforce',
    }

  },


  fn: async function ({salesforceAccountId, salesforceContactId, campaignName}) {

    // // Return undefined if we're not running in a production environment.
    // if(sails.config.environment !== 'production') {
    //   sails.log.verbose('Skipping Salesforce integration...');
    //   return {
    //     salesforceHistoricalEventId: undefined
    //   };
    // }

    require('assert')(sails.config.custom.salesforceIntegrationPasskey);
    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    let jsforce = require('jsforce');
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);
    let campaignRecord = await sails.helpers.flow.build(async ()=>{
      return await salesforceConnection.sobject('Campaign')
      .findOne({
        'Name':  campaignName
      });
    }).intercept((unusedErr)=>{
      throw 'campaignNotFound';
    });

    if(!campaignRecord || campaignRecord.Id) {
      throw 'campaignNotFound';
    }

    await sails.helpers.flow.build(async ()=>{
      return await salesforceConnection.sobject('CampaignMember')
      .create({
        Id: campaignRecord.Id,
        Account: salesforceAccountId,
        Contact: salesforceContactId,
      });
    }).intercept((err)=>{
      return new Error(`An error occured when creating a new "Campaign Member" record in Salesforce. full error ${require('util').inspect(err, {depth: null})}`);
    });


    // Note: This helper has no return value.
    return;

  }


};


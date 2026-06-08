module.exports = {


  friendlyName: 'Create campaign member',


  description: 'Creates a "Campaign Member" in Salesforce',


  inputs: {
    salesforceContactId: {
      type: 'string',
      required: true,
      extendedDescription: 'This ID of the contact that will be added to a campaign'
    },
    salesforceCampaignId: {
      type: 'string',
      required: true,
    },
  },


  exits: {

    success: {
      description: 'All done.',
    },

    couldNotCreateCampaignMember: {
      description: 'The provided campaign name did not match any active campaigns records in Salesforce',
    }

  },


  fn: async function ({salesforceContactId, salesforceCampaignId}) {

    // Stop running if we're not in a production environment.
    if(sails.config.environment !== 'production') {
      sails.log.verbose('Skipping Salesforce integration...');
      return;
    }

    require('assert')(sails.config.custom.salesforceIntegrationPasskey);
    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    let jsforce = require('jsforce');
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

    await sails.helpers.flow.build(async ()=>{
      return await salesforceConnection.sobject('CampaignMember')
      .create({
        CampaignId: salesforceCampaignId,
        ContactId: salesforceContactId,
      });
    }).intercept((err)=>{
      return new Error(`An error occured when creating a new "Campaign Member" record in Salesforce. full error ${require('util').inspect(err, {depth: null})}`);
    });

    // Note: This helper has no return value.
    return;

  }


};


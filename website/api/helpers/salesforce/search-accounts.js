module.exports = {


  friendlyName: 'Search accounts',


  description: 'Search Salesforce Account records by name.',


  inputs: {

    searchTerm: { type: 'string', required: true },

  },


  exits: {

    success: {
      outputFriendlyName: 'Matching accounts',
      outputType: [{
        salesforceAccountId: 'string',
        name: 'string',
        type: 'string',
        website: 'string',
      }],
    },

  },


  fn: async function ({searchTerm}) {

    if(sails.config.environment !== 'production') {
      sails.log.verbose('Skipping Salesforce integration...');
      return [];
    }

    require('assert')(sails.config.custom.salesforceIntegrationUsername);
    require('assert')(sails.config.custom.salesforceIntegrationPasskey);

    // jsforce escapes quotes for us, but LIKE wildcards need stripping so the search term is literal.
    let literalSearchTerm = searchTerm.replace(/[%_\\]/g, '').trim();
    if(!literalSearchTerm) {
      return [];
    }

    let jsforce = require('jsforce');
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);

    let accountRecords = await salesforceConnection.sobject('Account')
    .find({Name: {$like: `%${literalSearchTerm}%`}}, ['Id', 'Name', 'Type', 'Website'])
    .sort({Name: 1})
    .limit(10);

    return accountRecords.map((accountRecord)=>{
      return {
        salesforceAccountId: accountRecord.Id,
        name: accountRecord.Name,
        type: accountRecord.Type || '',
        website: accountRecord.Website || '',
      };
    });

  }


};

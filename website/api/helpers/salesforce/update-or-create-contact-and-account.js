module.exports = {


  friendlyName: 'Update or create contact and account',


  description: 'Upsert contact±account into Salesforce given fresh data about a particular person, and fresh IQ-enrichment data about the person and account.',


  inputs: {

    // Find by…
    emailAddress: { type: 'string' },
    linkedinUrl: { type: 'string' },

    // Set…
    firstName: { type: 'string' },
    lastName: { type: 'string' },
    organization: { type: 'string' },
    primaryBuyingSituation: { type: 'string' },
    psychologicalStage: { type: 'string' },
    numberOfHosts: { type: 'number' },

  },


  exits: {

    success: {
      outputType: {
        salesforceAccountId: 'string',
        salesforceContactId: 'string'
      }
    },

  },


  fn: async function ({emailAddress, linkedinUrl, firstName, lastName, organization, primaryBuyingSituation, psychologicalStage, numberOfHosts}) {
    require('assert')(sails.config.custom.salesforceSecret);// todo
    require('assert')(sails.config.custom.iqSecret);

    // TODO: require either emailAddress or linkedinUrl

    // TODO: enrich

    // TODO: update or create account and contact

    // TODO: return data
  }


};


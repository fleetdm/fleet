module.exports = {


  friendlyName: 'Search Salesforce accounts',


  description: 'Autocomplete company names from Salesforce when posting a role to fleetdm.com/hire.',


  inputs: {

    searchTerm: { type: 'string', required: true, minLength: 2, maxLength: 100 },

  },


  exits: {

    success: {
      outputType: [{}],
    },

  },


  fn: async function ({searchTerm}) {

    // Note: .intercept() can't be chained after .retry() (only the first matching handler runs), so use try/catch.
    let matchingAccounts;
    try {
      matchingAccounts = await sails.helpers.salesforce.searchAccounts.with({searchTerm}).retry();
    } catch (err) {
      throw new Error(`When a Fleet team member searched Salesforce accounts for the job board (search term: ${searchTerm}), an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
    }

    return matchingAccounts;

  }


};

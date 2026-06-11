module.exports = {


  friendlyName: 'Get territory user ID' ,


  description: 'Returns a Salesforce User ID who is associated with a location.',


  inputs: {
    state: { type: 'string' },
    city: { type: 'string' },
    country: { type: 'string', required: true},
  },


  exits: {

    success: {
      outputFriendlyName: 'territoryUserId',
    },

  },


  fn: async function ({state, city, country}) {
    //  ╦  ╔═╗╔═╗╦╔╗╔  ╔╦╗╔═╗  ╔═╗╔═╗╦  ╔═╗╔═╗╔═╗╔═╗╦═╗╔═╗╔═╗
    //  ║  ║ ║║ ╦║║║║   ║ ║ ║  ╚═╗╠═╣║  ║╣ ╚═╗╠╣ ║ ║╠╦╝║  ║╣
    //  ╩═╝╚═╝╚═╝╩╝╚╝   ╩ ╚═╝  ╚═╝╩ ╩╩═╝╚═╝╚═╝╚  ╚═╝╩╚═╚═╝╚═╝
    // Log in to Salesforce.
    let jsforce = require('jsforce');
    let salesforceConnection = new jsforce.Connection({
      loginUrl : 'https://fleetdm.my.salesforce.com'
    });
    await salesforceConnection.login(sails.config.custom.salesforceIntegrationUsername, sails.config.custom.salesforceIntegrationPasskey);
    // If the state is not set to California, remove the city (This is the only state that is in two different territories.)
    if(state && state.toLowerCase() !== 'california') {
      city = undefined;
    }
    let apexInputs = new URLSearchParams({state, country, city});

    let territoryInformation = await sails.helpers.flow.build(async ()=>{
      return await salesforceConnection.apex.get(`/territory-lookup?${apexInputs.toString()}`)
    }).intercept((err)=>{
      throw new Error(`When sending a request to Salesforce to lookup the territory ID for an address (${require('util').inspect({state, city, country})}) an error occured. Full error: ${require('util').inspect(err)}`)
    });
    if(!territoryInformation.users || !_.isArray(territoryInformation.users)) {
      throw new Error(`When looking up the territory ID for an address (${require('util').inspect({state, city, country})}), the information returned by Salesforce did not include a list of users. Territory information returned by Salesforce: ${require('util').inspect(territoryInformation)}`);
    } else if(!territoryInformation.users[0] || !territoryInformation.users[0].userId) {
      throw new Error(`When looking up the territory ID for an address (${require('util').inspect({state, city, country})}), the user information returned by Salesforce did not include the required information. Territory information returned by Salesforce: ${require('util').inspect(territoryInformation)}`);
    }

    let userIdForThisTeritory = territoryInformation.users[0].userId;
    // Send back the result through the success exit.
    return userIdForThisTeritory;

  }


};


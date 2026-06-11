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

    let apexInputs = new URLSearchParams({state, city, country});
    let territoryInformation = await sails.helpers.flow.build(async ()=>{
      return await salesforceConnection.apex.get(`/territory-lookup?${apexInputs.toString()}`)
    }).intercept((err)=>{
      throw new Error(`When sending a request to Salesforce to get lookup the territory ID for an address (${{state, city, country}}, an error occured. Full error: ${require('util').inspect(err)}`)
    });

    require('assert')(territoryInformation.users && _.isArray(territoryInformation.users));
    require('assert')(territoryInformation.users[0] && territoryInformation.users[0].userId);

    let userIdForThisTeritory = territoryInformation.users[0].userId;
    // Send back the result through the success exit.
    return userIdForThisTeritory;

  }


};


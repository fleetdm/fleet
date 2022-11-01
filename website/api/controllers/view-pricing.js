module.exports = {


  friendlyName: 'View pricing',


  description: 'Display "Pricing" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/pricing'
    },

    badConfig: {
      responseType: 'badConfig'
    },

  },


  fn: async function () {

    if(!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.pricingTable)) {
      throw {badConfig: 'builtStaticContent.'};
    }
    let pricingTable = sails.config.builtStaticContent.pricingTable;

    // Respond with view.
    return { pricingTable };

  }


};

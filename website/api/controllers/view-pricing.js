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
      throw {badConfig: 'builtStaticContent.pricingTable'};
    }
    let pricingTable = sails.config.builtStaticContent.pricingTable;

    // Create a filtered version of the pricing table array that does not have the "Device management" category that will be used for the security-focused pricing table.
    let pricingTableForSecurity = pricingTable.filter((category)=>{
      return category.categoryName !== 'Device management';
    });

    // Create an array used to sort the pricing table for secuirty focused buyers
    // To change the order of the pricing table for the security focused buyers, rearrange the values in the array below.
    // Note: The category names must match existing categories in the pricing-features-table.yml file.
    let categoryOrderForSecurityPricingTable = [
      'Security and compliance',
      'Monitoring',
      'Inventory management',
      'Collaboration',
      'Support',
      'Data outputs',
      'Deployment'
    ];

    // Sort the security-focused pricing table from the order of the elements in the categoryOrderForSecurityPricingTable array.
    pricingTableForSecurity.sort((a, b)=>{
      return categoryOrderForSecurityPricingTable.indexOf(a.categoryName) - categoryOrderForSecurityPricingTable.indexOf(b.categoryName);
    });

    // Respond with view.
    return {
      pricingTable,
      pricingTableForSecurity,
    };

  }


};

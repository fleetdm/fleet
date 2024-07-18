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
    let pricingTableFeatures = sails.config.builtStaticContent.pricingTable;

    let pricingTable = [];

    let pricingTableCategories = ['Device management', 'Support', 'Deployment', 'Integrations', 'Endpoint operations', 'Vulnerability management'];
    for(let category of pricingTableCategories) {
      // Get all the features in that have a pricingTableFeatures array that contains this category.
      let featuresInThisCategory = _.filter(pricingTableFeatures, (feature)=>{
        return _.contains(feature.pricingTableCategories, category);
      });
      // Build a dictionary containing the category name, and all features in the category, sorting premium features to the bottom of the list.
      let allFeaturesInThisCategory = {
        categoryName: category,
        features: featuresInThisCategory
      };
      // Add the dictionaries to the arrays that we'll use to build the features table.
      pricingTable.push(allFeaturesInThisCategory);
    }

    let pricingTableForSecurity = _.filter(pricingTable, (category)=>{
      return category.categoryName !== 'Device management' && (category.usualDepartment === 'Security' || category.usualDepartment === undefined);
    });
    let categoryOrderForSecurityPricingTable = ['Support', 'Deployment', 'Integrations', 'Endpoint operations', 'Vulnerability management'];
    // Sort the security-focused pricing table from the order of the elements in the categoryOrderForSecurityPricingTable array.
    pricingTableForSecurity.sort((a, b)=>{
      // If there is a category that is not in the list above, sort it to the end of the list.
      if(categoryOrderForSecurityPricingTable.indexOf(a.categoryName) === -1){
        return 1;
      } else if(categoryOrderForSecurityPricingTable.indexOf(b.categoryName) === -1) {
        return -1;
      }
      return categoryOrderForSecurityPricingTable.indexOf(a.categoryName) - categoryOrderForSecurityPricingTable.indexOf(b.categoryName);
    });


    let pricingTableForIt = _.filter(pricingTable, (category)=>{
      return category.categoryName !== 'Vulnerability management' && (category.usualDepartment === 'Security' || category.usualDepartment === undefined);
    });
    let categoryOrderForITPricingTable = ['Device management', 'Support', 'Deployment', 'Integrations', 'Endpoint operations'];
    // Sort the IT-focused pricing table from the order of the elements in the categoryOrderForITPricingTable array.
    pricingTableForIt.sort((a, b)=>{
      // If there is a category that is not in the list above, sort it to the end of the list.
      if(categoryOrderForITPricingTable.indexOf(a.categoryName) === -1){
        return 1;
      } else if(categoryOrderForITPricingTable.indexOf(b.categoryName) === -1) {
        return -1;
      }
      return categoryOrderForITPricingTable.indexOf(a.categoryName) - categoryOrderForITPricingTable.indexOf(b.categoryName);
    });


    // Respond with view.
    return {
      pricingTable,
      pricingTableForSecurity,
      pricingTableForIt
    };

  }


};

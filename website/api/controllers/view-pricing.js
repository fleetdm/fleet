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

    let pricingTableCategories = ['Devices', 'Deployment', 'Configuration', 'Integrations', 'Support'];
    for(let category of pricingTableCategories) {
      // Get all the features in that have a pricingTableFeatures array that contains this category.
      let featuresInThisCategory = _.filter(pricingTableFeatures, (feature)=>{
        return _.contains(feature.pricingTableCategories, category);
      });
      // Build a dictionary containing the category name, and all features in the category, sorting premium features to the bottom of the list.
      let allFeaturesInThisCategory = {
        categoryName: category,
        features: featuresInThisCategory,
      };
      // Add the dictionaries to the arrays that we'll use to build the features table.
      pricingTable.push(allFeaturesInThisCategory);
    }

    let pricingTableForSecurity = [];
    let categoryOrderForSecurityPricingTable = ['Devices', 'Deployment', 'Configuration', 'Integrations', 'Support'];
    for(let category of categoryOrderForSecurityPricingTable) {
      // Get all the features in that have a pricingTableFeatures array that contains this category.
      let featuresInThisCategory = _.filter(pricingTableFeatures, (feature)=>{
        return _.contains(feature.pricingTableCategories, category) && (feature.usualDepartment === 'Security' || feature.usualDepartment === undefined);
      });
      // Build a dictionary containing the category name, and all features in the category
      let allSecurityFeaturesInThisCategory = {
        categoryName: category,
        features: featuresInThisCategory,
      };
      // Add the dictionaries to the arrays that we'll use to build the features table.
      pricingTableForSecurity.push(allSecurityFeaturesInThisCategory);
    }


    let categoryOrderForITPricingTable = ['Devices', 'Deployment', 'Configuration', 'Integrations', 'Support'];
    let pricingTableForIt = [];
    // Sort the IT-focused pricing table from the order of the elements in the categoryOrderForITPricingTable array.
    for(let category of categoryOrderForITPricingTable) {
      // Get all the features in that have a pricingTableFeatures array that contains this category.
      let featuresInThisCategory = _.filter(pricingTableFeatures, (feature)=>{
        return _.contains(feature.pricingTableCategories, category) && (feature.usualDepartment === 'IT' || feature.usualDepartment === undefined);
      });
      // Build a dictionary containing the category name, and all features in the category, sorting premium features to the bottom of the list.
      let allItFeaturesInThisCategory = {
        categoryName: category,
        features: featuresInThisCategory,
      };
      // Add the dictionaries to the arrays that we'll use to build the features table.
      pricingTableForIt.push(allItFeaturesInThisCategory);
    }


    // Respond with view.
    return {
      pricingTable,
      pricingTableForSecurity,
      pricingTableForIt
    };

  }


};

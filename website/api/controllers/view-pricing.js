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

    // Create three separate arrays of features we'll use for our tables.
    // Note: We're build these arrays here instead of filtering the table in the frontend so we can render the tables on the server.
    let pricingTable = [];// For all features
    let securityPricingTable = [];// For features with usualDepartment: Security
    let itPricingTable = [];// For features with usualDepartment: IT

    // Note: These product categories are hardcoded in to reduce complexity, an alternative way of building this from the pricingFeaturesTable is: let productCategories =  _.union(_.flatten(_.pluck(pricingTableFeatures, 'productCategories')));
    let productCategories = ['Endpoint operations', 'Device management', 'Vulnerability management']
    for(let category of productCategories) {
      // Get all the features in that have a productCategories array that contains this category.
      let featuresInThisCategory = _.filter(pricingTableFeatures, (feature)=>{
        return _.contains(feature.productCategories, category);
      });
      // Filter the features in this category to build an array of the security-focused features.
      let securityFeaturesInThisCategory = _.filter(featuresInThisCategory, (feature)=>{
        return feature.usualDepartment === 'Security' || feature.usualDepartment === undefined;
      });
      // Filter the features in this category to build an array of the IT-focused features.
      let itFeaturesInThisCategory = _.filter(featuresInThisCategory, (feature)=>{
        return feature.usualDepartment === 'IT' || feature.usualDepartment === undefined;
      });
      // Build a dictionary containing the category name, and all features in the category, sorting premium features to the bottom of the list.
      let allFeaturesInThisCategory = {
        categoryName: category,
        features: _.sortBy(featuresInThisCategory, (feature)=>{
          return feature.tier !== 'Free';
        })
      };
      // Do the same thing for security-focused features
      let categoryForSecurity = {
        categoryName: category,
        features: _.sortBy(securityFeaturesInThisCategory, (feature)=>{
          return feature.tier !== 'Free';
        })
      };
      // And for IT-focused features.
      let categoryforIt = {
        categoryName: category,
        features: _.sortBy(itFeaturesInThisCategory, (feature)=>{
          return feature.tier !== 'Free';
        })
      };
      // Add the dictionaries to the arrays that we'll use to build the features table.
      pricingTable.push(allFeaturesInThisCategory);
      securityPricingTable.push(categoryForSecurity);
      itPricingTable.push(categoryforIt);
    }

    // Respond with view.
    return {
      pricingTable,
      pricingTableForSecurity: securityPricingTable,
      pricingTableForIt: itPricingTable
    };

  }


};

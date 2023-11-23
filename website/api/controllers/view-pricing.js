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

    // Note: These product categories are hardcoded in to reduce complexity, an alternative way of building this from the pricingFeaturesTable is: let productCategories =  _.union(_.flatten(_.pluck(pricingTableFeatures, 'productCategories')));
    let productCategories = ['Endpoint operations', 'Device management', 'Vulnerability management'];
    for(let category of productCategories) {
      // Get all the features in that have a productCategories array that contains this category.
      let featuresInThisCategory = _.filter(pricingTableFeatures, (feature)=>{
        return _.contains(feature.productCategories, category);
      });
      // Build a dictionary containing the category name, and all features in the category, sorting premium features to the bottom of the list.
      let allFeaturesInThisCategory = {
        categoryName: category,
        features: _.sortBy(featuresInThisCategory, (feature)=>{
          return feature.tier !== 'Free';
        })
      };
      // Add the dictionaries to the arrays that we'll use to build the features table.
      pricingTable.push(allFeaturesInThisCategory);
    }

    // Respond with view.
    return {
      pricingTable
    };

  }


};

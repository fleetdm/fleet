module.exports = {


  friendlyName: 'View android mdm',


  description: 'Display "Android MDM" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/android-mdm'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrollable-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Filter the testimonials to ones tagged with "Device management".
    testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Device management');
    });

    // Respond with view.
    return {
      testimonialsForScrollableTweets
    };

  }


};

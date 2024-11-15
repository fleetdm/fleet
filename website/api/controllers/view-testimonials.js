module.exports = {


  friendlyName: 'View testimonials',


  description: 'Display "Testimonials" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/testimonials'
    }

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonials = _.clone(sails.config.builtStaticContent.testimonials);

    // Filter the testimonials by product category
    let testimonialsForMdm = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Device management');
    });
    let testimonialsForSecurityEngineering = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Vulnerability management');
    });
    let testimonialsForItEngineering = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Endpoint operations');
    });
    let testimonialsWithVideoLinks = _.filter(testimonials, (testimonial)=>{
      return testimonial.youtubeVideoUrl;
    });

    return {
      testimonialsForMdm,
      testimonialsForSecurityEngineering,
      testimonialsForItEngineering,
      testimonialsWithVideoLinks,
    };

  }


};

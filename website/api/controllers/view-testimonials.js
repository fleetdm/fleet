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
    let testimonialOrderForMdm = [
      'Scott MacVicar',
      'Chandra Majumdar',
      'Nico Waisman',
      'Kenny Botelho',
      'Eric Tan',
      'Dan Grzelak',
      'Erik Gomez',
      'Matt Carr',
    ];
    let testimonialsForMdm = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Device management') && _.contains(testimonialOrderForMdm, testimonial.quoteAuthorName);
    });
    testimonialsForMdm.sort((a, b)=>{
      if(testimonialOrderForMdm.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForMdm.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForMdm.indexOf(a.quoteAuthorName) - testimonialOrderForMdm.indexOf(b.quoteAuthorName);
    });
    let testimonialOrderForSoftwareManagement = [
      'Wes Whetstone',
      'Erik Gomez',
      'Chandra Majumdar',
      'Kenny Botelho',
      'Arsenio Figueroa',
      'Andre Shields',
      'Nico Waisman',
      'Eric Tan',
      'Dan Grzelak',
    ];
    let testimonialsForSoftwareManagement = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Software management') && _.contains(testimonialOrderForSoftwareManagement, testimonial.quoteAuthorName);
    });
    testimonialsForSoftwareManagement.sort((a, b)=>{
      if(testimonialOrderForSoftwareManagement.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForSoftwareManagement.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForSoftwareManagement.indexOf(a.quoteAuthorName) - testimonialOrderForSoftwareManagement.indexOf(b.quoteAuthorName);
    });
    let testimonialOrderForObservability = [
      'Eric Tan',
      'Arsenio Figueroa',
      'Scott MacVicar',
      'Chandra Majumdar',
      'Kenny Botelho',
      'Brendan Shaklovitz',
      'Erik Gomez',
      'Charles Zaffery',
      'Ahmed Elshaer',
      'Andre Shields',
      'Mike Arpaia',
      'Tom Larkin',
    ];
    let testimonialsForObservability = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Observability') && _.contains(testimonialOrderForObservability, testimonial.quoteAuthorName);
    });
    testimonialsForObservability.sort((a, b)=>{
      if(testimonialOrderForObservability.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForObservability.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForObservability.indexOf(a.quoteAuthorName) - testimonialOrderForObservability.indexOf(b.quoteAuthorName);
    });
    let testimonialsWithVideoLinks = _.filter(testimonials, (testimonial)=>{
      return testimonial.youtubeVideoUrl;
    });

    return {
      testimonialsForMdm,
      testimonialsForSoftwareManagement,
      testimonialsForObservability,
      testimonialsWithVideoLinks,
    };

  }


};

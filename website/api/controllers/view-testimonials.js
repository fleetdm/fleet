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
    let testimonialOrderForMdm = [
      'Scott MacVicar',
      'Wes Whetstone',
      'Nick Fohs',
      'Erik Gomez',
      'Matt Carr',
      'Nico Waisman',
      'Kenny Botelho',
      'Dan Grzelak',
      'Eric Tan',
    ];
    testimonialsForMdm.sort((a, b)=>{
      if(testimonialOrderForMdm.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForMdm.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForMdm.indexOf(a.quoteAuthorName) - testimonialOrderForMdm.indexOf(b.quoteAuthorName);
    });
    let testimonialsForSecurityEngineering = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Vulnerability management');
    });
    let testimonialOrderForSecurityEngineering = [
      'Nico Waisman',
      'Austin Anderson',
      'Chandra Majumdar',
      'Andre Shields',
      'Dan Grzelak',
      'Charles Zaffery',
      'Erik Gomez',
      'Nick Fohs',
      'Dhruv Majumdar',
      'Arsenio Figueroa',
    ];
    testimonialsForSecurityEngineering.sort((a, b)=>{
      if(testimonialOrderForSecurityEngineering.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForSecurityEngineering.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForSecurityEngineering.indexOf(a.quoteAuthorName) - testimonialOrderForSecurityEngineering.indexOf(b.quoteAuthorName);
    });
    let testimonialsForItEngineering = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Endpoint operations');
    });
    let testimonialOrderForItEngineering = [
      'Charles Zaffery',
      'Nico Waisman',
      'Erik Gomez',
      'Mike Arpaia',
      'Ahmed Elshaer',
      'Kenny Botelho',
      'Alvaro Gutierrez',
      'Tom Larkin',
      'Nick Fohs',
      'Andre Shields',
      'Abubakar Yousafzai',
      'Chandra Majumdar',
      'Joe Pistone',
      'Dan Grzelak',
      'Austin Anderson',
      'Brendan Shaklovitz',
      'Dhruv Majumdar',
      'Wes Whetstone',
      'Eric Tan',
      'Arsenio Figueroa',
    ];
    testimonialsForItEngineering.sort((a, b)=>{
      if(testimonialOrderForItEngineering.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForItEngineering.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForItEngineering.indexOf(a.quoteAuthorName) - testimonialOrderForItEngineering.indexOf(b.quoteAuthorName);
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

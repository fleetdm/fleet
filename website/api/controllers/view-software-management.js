module.exports = {


  friendlyName: 'View software-management',


  description: 'Display "Software management" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/software-management'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Specify an order for the testimonials on this page using the last names of quote authors
    let testimonialOrderForThisPage = [
      'Luis Madrigal',
      'Arsenio Figueroa',
      'Bart Reardon',
      'Andre Shields',
      'Wes Whetstone',
      'Nico Waisman',
      'Chandra Majumdar',
      'Kenny Botelho',
      'Erik Gomez',
      'Eric Tan',
      'Dan Grzelak',
      'Justin LaBo'
    ];

    // Filter the testimonials by product category
    testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Software management') && _.contains(testimonialOrderForThisPage, testimonial.quoteAuthorName);
    });

    testimonialsForScrollableTweets.sort((a, b)=>{
      if(testimonialOrderForThisPage.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForThisPage.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForThisPage.indexOf(a.quoteAuthorName) - testimonialOrderForThisPage.indexOf(b.quoteAuthorName);
    });

    // Respond with view.
    return {
      testimonialsForScrollableTweets,
    };

  }


};

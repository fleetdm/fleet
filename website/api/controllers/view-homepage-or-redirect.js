module.exports = {


  friendlyName: 'View homepage or redirect',


  description: 'Display or redirect to the appropriate homepage, depending on login status.',


  exits: {

    success: {
      statusCode: 200,
      description: 'Requesting user is a guest, so show the public landing page.',
      viewTemplatePath: 'pages/homepage'
    },

    redirect: {
      responseType: 'redirect',
      description: 'Requesting user is logged in, so redirect to the internal welcome page.'
    },

    badConfig: {
      responseType: 'badConfig'
    },

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Specify an order for the testimonials on this page using the last names of quote authors
    let testimonialOrderForThisPage = [
      'Bart Reardon',
      'Scott MacVicar',
      'Mike Meyer',
      'Luis Madrigal',
      'Tom Larkin',
      'Kenny Botelho',
      'Erik Gomez',
      'Chandra Majumdar',
      'Eric Tan',
      'Matt Carr',
      'Nico Waisman',
      'Dan Grzelak',
      'Philip Chotipradit',
      'Roger Cantrell',
      'Chayce O\'Neal',
      'David Bodmer'
    ];

    // Filter the testimonials by product category
    testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Device management') && _.contains(testimonialOrderForThisPage, testimonial.quoteAuthorName);
    });

    testimonialsForScrollableTweets.sort((a, b)=>{
      if(testimonialOrderForThisPage.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForThisPage.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForThisPage.indexOf(a.quoteAuthorName) - testimonialOrderForThisPage.indexOf(b.quoteAuthorName);
    });


    return {
      testimonialsForScrollableTweets
    };

  }


};

module.exports = {


  friendlyName: 'View windows management',


  description: 'Display "Windows management" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/windows-management'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrollable-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Specify an order for the testimonials on this page using the names of quote authors.
    // Surface multi-OS and Windows-relevant voices first.
    let testimonialOrderForThisPage = [
      'Justin LaBo',
      'u/Heteronymous',
      'Fiona Skelton',
      'Dan Jackson',
      'Nick Fohs',
      'Erik Gomez',
      'Dan Grzelak',
      'matt carr',
      'Bart Reardon',
      'David Bodmer',
      'Roger Cantrell',
    ];

    // Filter the testimonials by product category.
    testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Device management');
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
      testimonialsForScrollableTweets
    };

  }


};

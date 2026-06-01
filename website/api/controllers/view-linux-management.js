module.exports = {


  friendlyName: 'View linux management',


  description: 'Display "Linux management" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/linux-management'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials)) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Only filter and sort testimonials when static content has been built.
    // If the build-static-content script was not run, we'll show a placeholder testimonial that is added by the custom hook.
    if (sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      // Specify an order for the testimonials on this page using the last names of quote authors
      let testimonialOrderForThisPage = [
        'Erik Gomez',
        'Nick Fohs',
        'Dan Grzelak',
        'matt carr',
        'Justin LaBo',
        'Bart Reardon',
        'Dan Jackson',
        'David Bodmer',
        'Fiona Skelton',
        'Roger Cantrell',
        'u/Heteronymous',
      ];

      // Filter the testimonials by product category
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
    }//ﬁ

    // Respond with view.
    return {
      testimonialsForScrollableTweets
    };

  }


};

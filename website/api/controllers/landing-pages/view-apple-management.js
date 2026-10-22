module.exports = {


  friendlyName: 'View apple management',


  description: 'Display "apple management" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/apple-management'
    },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials)) {
      throw new Error('Could not build the page: sails.config.builtStaticContent.testimonials is missing. Run `sails run build-static-content` to generate it.');
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Only filter and sort testimonials when static content has been built.
    // If the build-static-content script was not run, we'll show a placeholder testimonial that is added by the custom hook.
    if (sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      // Specify an order for the testimonials on this page using the last names of quote authors
      let testimonialOrderForThisPage = [
        'Mike Meyer',
        'Wes Whetstone',
        'Erik Gomez',
        'Chayce O\'Neal',
        'Nick Fohs',
        'Kenny Botelho',
        'Dan Jackson',
        'Bart Reardon',
        'Eric Tan',
        'Luis Madrigal',
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

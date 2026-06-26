module.exports = {


  friendlyName: 'View security and control',


  description: 'Display "Security and control" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/security-and-control'
    }

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
      // Note: this page uses the same testimonials as the /software-managment page.
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
        'Adam Pippert',
        'Justin LaBo',
        'Brian LaShomb',
        'Ed Merrett',
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
    }//ﬁ


    // Respond with view.
    return {
      testimonialsForScrollableTweets,
    };


  }


};

module.exports = {


  friendlyName: 'View apple mdm',


  description: 'Display "Apple MDM" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/apple-mdm'
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
    let testimonialOrderForThisPage = [
      'Mike Meyer',
      'Wes Whetstone',
      'Erik Gomez',
      'Thomas Lübker',
      'Nick Fohs',
      'Chayce O\'Neal',
      'Luis Madrigal',
      'Kenny Botelho',
      'Dan Jackson',
      'Bart Reardon',
      'John O\'Nolan',
      'Eric Tan',
    ];

    // Filter the testimonials to ones tagged with "Device management".
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

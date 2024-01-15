module.exports = {


  friendlyName: 'View fleet mdm',


  description: 'Display "Fleet mdm" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/fleet-mdm'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Filter the testimonials by product category
    testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Device management');
    });

    // Specify an order for the testimonials on this page using the last names of quote authors
    let testimonialOrderForThisPage = ['Gomez', 'Fohs', 'Grzelak', 'Botelho', 'Whetstone', 'Carr'];
    testimonialsForScrollableTweets.sort((a, b)=>{
      if(testimonialOrderForThisPage.indexOf(a.quoteAuthorName.split(' ')[1]) === -1){
        return 1;
      } else if(testimonialOrderForThisPage.indexOf(b.quoteAuthorName.split(' ')[1]) === -1) {
        return -1;
      }
      return testimonialOrderForThisPage.indexOf(a.quoteAuthorName.split(' ')[1]) - testimonialOrderForThisPage.indexOf(b.quoteAuthorName.split(' ')[1]);
    });

    // Respond with view.
    return {
      testimonialsForScrollableTweets,
    };

  }


};

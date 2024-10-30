module.exports = {


  friendlyName: 'View device management',


  description: 'Display "Device management" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/device-management'
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
    let testimonialOrderForThisPage = ['Scott MacVicar', 'Erik Gomez', 'Kenny Botelho', 'Wes Whetstone', 'Matt Carr', 'Dan Grzelak', 'Nick Fohs'];
    testimonialsForScrollableTweets.sort((a, b)=>{
      if(testimonialOrderForThisPage.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForThisPage.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForThisPage.indexOf(a.quoteAuthorName) - testimonialOrderForThisPage.indexOf(b.quoteAuthorName);
    });

    let showSwagForm = false;
    // Due to shipping costs, we'll check the requesting user's cf-ipcountry to see if they're in the US, and their cf-iplongitude header to see if they're in the contiguous US.
    if(this.req.get('cf-ipcountry') === 'US' && this.req.get('cf-iplongitude') > -125) {
      showSwagForm = true;
    }

    // Respond with view.
    return {
      testimonialsForScrollableTweets,
      showSwagForm,
    };

  }


};

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

    // Specify an order for the testimonials on this page using the last names of quote authors
    let testimonialOrderForThisPage = [
      'Scott MacVicar',
      'Kenny Botelho',
      'Erik Gomez',
      'Chandra Majumdar',
      'Eric Tan',
      'Matt Carr',
      'Nico Waisman',
      'Dan Grzelak',
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

    let showSwagForm = false;
    // Due to shipping costs, we'll check the requesting user's cf-ipcountry to see if they're in the US, and their cf-iplongitude header to see if they're in the contiguous US.
    if(sails.config.environment === 'production') {
      // Log a warning if the cloudflare headers we use are missing in production.
      if(!this.req.get('cf-ipcountry') || !this.req.get('cf-iplongitude')) {
        sails.log.warn('When a user visted the device management page, the Cloudflare header we use to determine if they are visiting from the contiguous United States is missing.');
      }
    }
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

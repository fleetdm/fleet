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

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials)) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    let testimonials = _.clone(sails.config.builtStaticContent.testimonials);

    // Only filter and sort testimonials when static content has been built.
    // If the build-static-content script was not run, we'll show a placeholder testimonial that is added by the custom hook.
    if (sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      // The testimonials are shown left to right in this order: well-known brands first, with the Stripe quotes spread out.
      // Note: A name that appears more than once in this list matches that author's quotes in the order they appear in testimonials.yml.
      let sortOrderOfTestimonialAuthorsShownOnThisPage = [
        'Kenny Botelho',// NVIDIA
        'Scott MacVicar',// Stripe
        'Erik Gomez',// Uber
        'Dan Grzelak',// Atlassian
        'Brian LaShomb',// Epic Games
        'nico waisman',// Lyft
        'Andre Shields',// Rivian
        'Roger Cantrell',// Nutanix
        'Andy Gombar',// Webflow
        'Adam Pippert',// Red Hat
        'Brad Girardeau',// Figma
        'Ahmed Elshaer',// Salesforce
        'Adam Anklewicz',// Thumbtack
        'Nick Fohs',
        'Wes Whetstone',// Stripe
        'Dan Jackson',// Fastly
        'Eric Tan',// Flock Safety
        'Viktor Filipsson',// Sonos
        'Luis Madrigal',// Uber
        'Mike Arpaia',// osquery
        'Matt Carr',
        'Justin LaBo',// Benchling
        'Austin Anderson',
        'Chayce O\'Neal',// ConductorOne
        'Mike Meyer',// Foursquare
        'Wes Whetstone',// Stripe
      ];

      let testimonialAuthorsToExcludeOnThisPage = [
        'Alvaro Gutierrez',
        'Joe Pistone',
        'Brendan Shaklovitz',
        'Abubakar Yousafzai',
        'Dhruv Majumdar',
        'matt carr',
        'Charles Zaffery',
        'Tom Larkin',// Note: excluded becasue we already show a quote from this person
        'Nico Waisman',// Note: excluded becasue we already show a quote from this person
      ];

      testimonials = _.filter(testimonials, (testimonial)=>{
        return !testimonialAuthorsToExcludeOnThisPage.includes(testimonial.quoteAuthorName);
      });

      // Put the testimonials in the order above, followed by any others in the order they are listed in testimonials.yml.
      let remainingTestimonials = _.clone(testimonials);
      let sortedTestimonials = [];
      for(let authorName of sortOrderOfTestimonialAuthorsShownOnThisPage) {
        let indexOfNextQuote = _.findIndex(remainingTestimonials, {quoteAuthorName: authorName});
        if(indexOfNextQuote !== -1) {
          sortedTestimonials.push(remainingTestimonials.splice(indexOfNextQuote, 1)[0]);
        }
      }//∞
      testimonials = sortedTestimonials.concat(remainingTestimonials);
    }//ﬁ

    let testimonialsWithVideoLinks = _.filter(testimonials, (testimonial)=>{
      return testimonial.youtubeVideoUrl;
    });

    return {
      testimonials,
      testimonialsWithVideoLinks,
    };

  }


};

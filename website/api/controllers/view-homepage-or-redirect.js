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
      // Same sort order and exclusions as the /customers page (see view-customers.js).
      let sortOrderOfTestimonialAuthorsShownOnThisPage = [
        'Kenny Botelho',
        'Scott MacVicar',
        'Adam Anklewicz',
        'Nick Fohs',
        'Wes Whetstone',
        'Erik Gomez',
        'Matt Carr',
        'nico waisman',
        'Dan Grzelak',
        'Eric Tan',
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

      testimonials.sort((a, b)=>{
        if(sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(a.quoteAuthorName) === -1){
          return 1;
        } else if(sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(b.quoteAuthorName) === -1) {
          return -1;
        }
        return sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(a.quoteAuthorName) - sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(b.quoteAuthorName);
      });
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

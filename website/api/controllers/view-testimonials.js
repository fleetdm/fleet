module.exports = {


  friendlyName: 'View testimonials',


  description: 'Display "Testimonials" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/testimonials'
    }

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the page contents
    let testimonials = _.clone(sails.config.builtStaticContent.testimonials);

    // Filter the testimonials by product category
    let testimonialOrderForMdm = [
      'Bart Reardon',
      'Scott MacVicar',
      'Mike Meyer',
      'Tom Larkin',
      'Kenny Botelho',
      'Erik Gomez',
      'Chandra Majumdar',
      'Eric Tan',
      'Matt Carr',
      'Nico Waisman',
      'Dan Grzelak',
      'Philip Chotipradit',
      'Roger Cantrell',
      'Chayce O\'Neal',
      'u/Heteronymous',
    ];
    let testimonialsForMdm = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Device management') && _.contains(testimonialOrderForMdm, testimonial.quoteAuthorName);
    });
    testimonialsForMdm.sort((a, b)=>{
      if(testimonialOrderForMdm.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForMdm.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForMdm.indexOf(a.quoteAuthorName) - testimonialOrderForMdm.indexOf(b.quoteAuthorName);
    });
    let testimonialOrderForSoftwareManagement = [
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
      'Dan Grzelak',
    ];
    let testimonialsForSoftwareManagement = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Software management') && _.contains(testimonialOrderForSoftwareManagement, testimonial.quoteAuthorName);
    });
    testimonialsForSoftwareManagement.sort((a, b)=>{
      if(testimonialOrderForSoftwareManagement.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForSoftwareManagement.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForSoftwareManagement.indexOf(a.quoteAuthorName) - testimonialOrderForSoftwareManagement.indexOf(b.quoteAuthorName);
    });
    let testimonialOrderForObservability = [
      'Ahmed Elshaer',
      'Brendan Shaklovitz',
      'Arsenio Figueroa',
      'Luis Madrigal',
      'Andre Shields',
      'Tom Larkin',
      'Matt Carr',
      'Eric Tan',
      'Charles Zaffery',
      'Kenny Botelho',
      'Scott MacVicar',
      'Erik Gomez',
      'Mike Arpaia',
      'Chandra Majumdar',
      'Justin LaBo',
      'tom larkin',
    ];
    let testimonialsForObservability = _.filter(testimonials, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Observability') && _.contains(testimonialOrderForObservability, testimonial.quoteAuthorName);
    });
    testimonialsForObservability.sort((a, b)=>{
      if(testimonialOrderForObservability.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForObservability.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForObservability.indexOf(a.quoteAuthorName) - testimonialOrderForObservability.indexOf(b.quoteAuthorName);
    });
    let testimonialsWithVideoLinks = _.filter(testimonials, (testimonial)=>{
      return testimonial.youtubeVideoUrl;
    });

    // Get articles with a showOnTestimonialsPageWithEmoji meta tag to display on this page.
    let articles = sails.config.builtStaticContent.markdownPages.filter((page)=>{
      if(_.startsWith(page.htmlId, 'articles')) {
        return page;
      }
    });
    let articlesForThisPage = _.filter(articles, (article)=>{
      return article.meta.showOnTestimonialsPageWithEmoji;
    });
    // Sort the articles by their publish date.
    articlesForThisPage = _.sortBy(articlesForThisPage, 'meta.publishedOn');




    return {
      testimonialsForMdm,
      testimonialsForSoftwareManagement,
      testimonialsForObservability,
      testimonialsWithVideoLinks,
      articlesForThisPage,
    };

  }


};

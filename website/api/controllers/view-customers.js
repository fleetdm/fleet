module.exports = {


  friendlyName: 'View customers',


  description: 'Display "Customers" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/customers'
    }

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }
    // Get testimonials for the page contents
    let testimonials = _.clone(sails.config.builtStaticContent.testimonials);


    let sortOrderOfTestimonialAuthorsShownOnThisPage = [
      'Scott MacVicar',
      'Adam Anklewicz',
      'Nick Fohs',
      'Wes Whetstone',
      'Erik Gomez',
      'Matt Carr',
      'nico waisman',
      'Kenny Botelho',
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

    let filteredTestimonialsForThisPage = _.filter(testimonials, (testimonial)=>{
      return !testimonialAuthorsToExcludeOnThisPage.includes(testimonial.quoteAuthorName);
    });

    filteredTestimonialsForThisPage.sort((a, b)=>{
      if(sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(a.quoteAuthorName) - sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(b.quoteAuthorName);
    });

    let testimonialsWithVideoLinks = _.filter(filteredTestimonialsForThisPage, (testimonial)=>{
      return testimonial.youtubeVideoUrl;
    });

    // Get all of the case study articles.
    let caseStudies = sails.config.builtStaticContent.markdownPages.filter((page)=>{
      if(_.startsWith(page.url, '/case-study/')) {
        return page;
      }
    });

    // Only show case studies that have `useBasicArticleTemplate` and cardTitleForCustomersPage` meta tags
    let caseStudiesToCreateLinksFor = caseStudies.filter((article)=>{
      if(article.meta.useBasicArticleTemplate && article.meta.cardTitleForCustomersPage){
        return article;
      }
    });
    // Sort the case study articles by the lowercase cardTitleForCustomersPage meta tag value.
    caseStudiesToCreateLinksFor = _.sortBy(caseStudiesToCreateLinksFor, (article)=>{
      return article.meta.cardTitleForCustomersPage.toLowerCase();
    });

    // We want to show 12 links to case studies initially, so we'll find out how many cards we need to show when the page loads
    let sizeOfFristBlockOfCaseStudyCards = 12 - (caseStudies.length - caseStudiesToCreateLinksFor.length);
    // And remove that many case studies from the array of case studies to create links for.
    let initalCaseStudyLinksToShow = caseStudiesToCreateLinksFor.splice(0, sizeOfFristBlockOfCaseStudyCards);
    // And we'll break the rest of the array into smaller arrays of 12 case studies.
    let blocksOfCaseStudies = _.chunk(caseStudiesToCreateLinksFor, 12);
    let numberOfPagesOfCaseStudies = blocksOfCaseStudies.length + 1;
    return {
      testimonials: filteredTestimonialsForThisPage,
      testimonialsWithVideoLinks,
      caseStudiesToCreateLinksFor,
      initalCaseStudyLinksToShow,
      blocksOfCaseStudies,
      numberOfPagesOfCaseStudies
    };

  }


};

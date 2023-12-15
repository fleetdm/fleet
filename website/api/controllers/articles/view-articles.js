module.exports = {


  friendlyName: 'View articles',


  description: 'Display "Articles" page.',


  inputs: {
    category: {
      type: 'string',
      description: 'The category of article to display.',
      defaultsTo: '',
    }
  },


  exits: {

    success: { viewTemplatePath: 'pages/articles/articles' },
    badConfig: { responseType: 'badConfig' },
    notFound: { responseType: 'notFound' },
    redirect: { responseType: 'redirect' },

  },


  fn: async function ({category}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }
    let articles = [];
    if (category === 'articles') {
      // If the category is `/articles` we'll show all articles
      articles = sails.config.builtStaticContent.markdownPages.filter((page)=>{
        if(_.startsWith(page.htmlId, 'articles')) {
          return page;
        }
      });
      // setting the category to all
      category = 'all';
    } else {
      // if the user navigates to a URL for a specific category, we'll only display articles in that category
      articles = sails.config.builtStaticContent.markdownPages.filter((page)=>{
        if(_.startsWith(page.url, '/'+category)) {
          return page;
        }
      });
    }

    let pageTitleForMeta = 'Fleet blog | Fleet';
    let pageDescriptionForMeta = 'Read the latest articles written by Fleet.';
    // Create a currentSection variable, this will be used to highlight the header dropdown that this article category lives under.
    // There are three possible values for this (documentation, community, and platform), so we'll default to the one with the most article categories (community) and set the value to another section if needed.
    // If the category is deploy, guides, or releases, currentSection will be set to 'documentation', and if the category is 'success-stories', currentSection will be set to 'platform'.
    let currentSection = 'community';

    // Set a pageTitleForMeta, pageDescriptionForMeta, and currentSection variable based on the article category.
    switch(category) {
      case 'success-stories':
        pageTitleForMeta = 'Success stories | Fleet';
        pageDescriptionForMeta = 'Read about how others are using Fleet and osquery.';
        currentSection = 'platform';
        break;
      case 'deploy':
        pageTitleForMeta = 'Deployment guides | Fleet';
        pageDescriptionForMeta = 'Learn how to deploy Fleet on a variety of production environments.';
        currentSection = 'documentation';
        break;
      case 'releases':
        pageTitleForMeta = 'Releases | Fleet';
        pageDescriptionForMeta = 'Fleet releases new and updated features every three weeks. Read about the latest product improvements here.';
        currentSection = 'documentation';
        break;
      case 'guides':
        pageTitleForMeta = 'Guides | Fleet';
        pageDescriptionForMeta = 'A collection of how-to guides for Fleet and osquery.';
        currentSection = 'documentation';
        break;
      case 'securing':
        pageTitleForMeta = 'Security articles | Fleet';
        pageDescriptionForMeta = 'Learn more about how we secure Fleet.';
        break;
      case 'engineering':
        pageTitleForMeta = 'Engineering articles | Fleet';
        pageDescriptionForMeta = 'Read about engineering at Fleet and beyond.';
        break;
      case 'announcements':
        pageTitleForMeta = 'Announcements | Fleet';
        pageDescriptionForMeta = 'Read the latest news from Fleet.';
        break;
      case 'podcasts':
        pageTitleForMeta = 'Podcasts | Fleet';
        pageDescriptionForMeta = 'Listen to the Future of Device Management podcast.';
        break;
    }


    return {
      path: require('path'),
      articles,
      category,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath,
      currentSection,
      pageTitleForMeta,
      pageDescriptionForMeta,
    };

  }


};

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
    // Set a currentSection variable for the website header based on how the articles category page is linked to in the header navigation dropdown menus.
    let currentSection;
    if(category === 'success-stories'){
      // If the category is success-stories, highlight the "Platform" dropdown.
      currentSection = 'platform';
    } else if(_.contains(['deploy','guides','releases'], category)) {
      // If the category is deploy, guides, or release, highlight the "Documentation" dropdown.
      currentSection = 'documentation';
    } else {
      // If the category is anything else, highlight the "Community" dropdown.
      currentSection = 'community';
    }

    return {
      path: require('path'),
      articles,
      category,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath,
      currentSection,
    };

  }


};

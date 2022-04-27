module.exports = {


  friendlyName: 'View articles',


  description: 'Display "Articles" page.',

  inputs: {
    category: {
      type: 'string',
      required: false,
      description: 'The category of article to display',
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
    if (category === '' || category === 'articles') {
      // If no category was provided (TODO: why would this happen?) or the user navigates to `/articles`, we'll show all articles
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

    return {
      path: require('path'),
      articles,
      category,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath,
    };

  }


};

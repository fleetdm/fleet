module.exports = {


  friendlyName: 'View articles',


  description: 'Display "Articles" page.',


  exits: {

    success: { viewTemplatePath: 'pages/articles/articles' },
    badConfig: { responseType: 'badConfig' },
    notFound: { responseType: 'notFound' },
    redirect: { responseType: 'redirect' },

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }
    let articles = [];

    articles = sails.config.builtStaticContent.markdownPages.filter((page)=>{
      if(_.startsWith(page.url, '/articles')) {
        return page;
      }
    });
    console.log(articles);

    return {
      path: require('path'),
      articles,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath,
    };

  }


};

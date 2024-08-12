module.exports = {

  friendlyName: 'View blog article',


  description: 'Display "Blog article" page.',


  urlWildcardSuffix: 'pageUrlSuffix',


  inputs: {
    pageUrlSuffix : {
      description: 'The relative path to the blog article page from within this route.',
      example: 'guides/deploying-fleet-on-render',
      type: 'string',
      defaultsTo: ''
    }
  },


  exits: {
    success: { viewTemplatePath: 'pages/articles/basic-article' },
    badConfig: { responseType: 'badConfig' },
    notFound: { responseType: 'notFound' },
    redirect: { responseType: 'redirect' },
  },


  fn: async function ({pageUrlSuffix}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }

    // Serve appropriate page content.
    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, { url: '/' + pageUrlSuffix });
    if (!thisPage) {// If there's no EXACTLY matching content page, try a revised version of the URL suffix that's lowercase, with all slashes deduped, and any leading or trailing slash removed (leading slashes are only possible if this is a regex, rather than "/*" route)
      let revisedPageUrlSuffix = pageUrlSuffix.toLowerCase().replace(/\/+/g, '/').replace(/^\/+/,'').replace(/\/+$/,'');
      thisPage = _.find(sails.config.builtStaticContent.markdownPages, { url: '/' + revisedPageUrlSuffix });
      if (thisPage) {// If we matched a page with the revised suffix, then redirect to that rather than rendering it, so the URL gets cleaned up.
        throw {redirect: thisPage.url};
      } else {// If no page could be found even with the revised suffix, then throw a 404 error.
        throw 'notFound';
      }
    }

    let articleCategorySlug = pageUrlSuffix.split('/')[0];

    // Setting the pages meta title and description from the articles meta tags, as well as an article image, if provided.
    // Note: Every article page should have a 'articleTitle' and a 'authorFullName' meta tag.
    // Note: Leaving title and description as `undefined` in our view means we'll default to the generic title and description set in layout.ejs.
    let pageTitleForMeta;
    if(thisPage.meta.articleTitle) {
      pageTitleForMeta = thisPage.meta.articleTitle;
    }//ﬁ
    let pageDescriptionForMeta;
    if(thisPage.meta.description){
      pageDescriptionForMeta = thisPage.meta.description;
    } else if(thisPage.meta.articleTitle && thisPage.meta.authorFullName) {
      pageDescriptionForMeta = _.trimRight(thisPage.meta.articleTitle, '.') + ' by ' + thisPage.meta.authorFullName;
    }//ﬁ

    // Set a currentSection variable for the website header based on how the articles category page is linked to in the header navigation dropdown menus.
    let currentSection;
    if(articleCategorySlug === 'success-stories'){
      // If the article is in the 'device-management' category, highlight the "Platform" dropdown.
      currentSection = 'platform';
    } else if(_.contains(['deploy','guides','releases'], articleCategorySlug)) {
      // If the articleCategorySlug is deploy, guides, or release, highlight the "Documentation" dropdown.
      currentSection = 'documentation';
    } else {
      // If the article is in any other category, highlight the "Community" dropdown.
      currentSection = 'community';
    }


    // Respond with view.
    return {
      path: require('path'),
      thisPage: thisPage,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath,
      pageTitleForMeta,
      pageDescriptionForMeta,
      pageImageForMeta: thisPage.meta.articleImageUrl || undefined,
      articleCategorySlug,
      currentSection,
    };

  }


};

module.exports = {

  friendlyName: 'View blog article',


  description: 'Display "Blog article" page.',

  urlWildcardSuffix: 'slug',

  inputs: {
    slug : {
      description: 'The relative path to the blog page from within this route.',
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


  fn: async function ({slug}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }
    
    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, { url: _.trimRight('/' + slug) });
    if (!thisPage) {// If there's no matching page, try a revised version of the slug that's lowercase, with internal slashes deduped, and any trailing slash trimmed
      let revisedSlug = slug.toLowerCase().replace(/\/+/g, '/').replace(/\/+$/,'');
      let revisedPage = _.find(sails.config.builtStaticContent.markdownPages, { url: _.trimRight('/' + revisedSlug) });
      if(revisedPage) {// If we matched a page with the revised slug, then redirect to that rather than rendering it, so the URL gets cleaned up.
        throw {redirect: revisedPage.url};
      } else {// If no page could be found even with the revised slug, then throw a 404 error.
        throw 'notFound';
      }
    }
    
    // Setting the pages meta title and description from the articles meta tags.
    // Note: Every article page will have a 'articleTitle' and a 'authorFullName' meta tag.
    // if they are undefined, we'll use the generic title and description set in layout.ejs
    let pageTitleForMeta;
    if(thisPage.meta.articleTitle) {
      pageTitleForMeta = thisPage.meta.articleTitle + ' | Fleet for osquery';
    }
    let pageDescriptionForMeta;
    if(thisPage.meta.articleTitle && thisPage.meta.authorFullName) {
      pageDescriptionForMeta = _.trimRight(thisPage.meta.articleTitle, '.') +' by '+thisPage.meta.authorFullName;
    }
    let pageImageForMeta;
    if(thisPage.meta.articleImageUrl){
      pageImageForMeta = thisPage.meta.articleImageUrl;
    }

    // Respond with view.
    return {
      path: require('path'),
      thisPage: thisPage,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath,
      pageTitleForMeta,
      pageDescriptionForMeta,
      pageImageForMeta,
    };

  }


};

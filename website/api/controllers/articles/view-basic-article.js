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
    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, {
      url: _.trimRight('/' + slug)
    });
    let needsRedirectMaybe = (!thisPage);

    if (needsRedirectMaybe) {
      // Creating a lower case, repeating-slashless slug
      let multipleSlashesRegex = /\/{2,}/g;
      let modifiedslug = slug.toLowerCase().replace(multipleSlashesRegex, '/');
      // Finding the appropriate page content using the modified slug.
      let revisedPage = _.find(sails.config.builtStaticContent.markdownPages, {
        url: _.trimRight('/' + _.trim(modifiedslug, '/'), '/')
      });
      if(revisedPage) {
        // If we matched a page with the modified slug, then redirect to that.
        throw {redirect: revisedPage.url};
      } else {
        // If no page was found, throw a 404 error.
        throw 'notFound';
      }
    }
    // Setting the pages meta title and description from the articles meta tags.
    // Note: Every article page will have a 'articleTitle' and a 'authorsFullName' meta tag.
    // if they are undefined, we'll use the generic title and description set in layout.ejs
    let pageTitleForMeta;
    if(thisPage.meta.articleTitle) {
      pageTitleForMeta = thisPage.meta.articleTitle + ' | Fleet for osquery';
    }
    let pageDescriptionForMeta;
    if(!thisPage.meta.articleTitle || !thisPage.meta.authorsFullName) {
      pageDescriptionForMeta = thisPage.meta.articleTitle +' by '+thisPage.meta.authorsFullName+'.';
    }

    // Respond with view.
    return {
      path: require('path'),
      thisPage: thisPage,
      markdownPages: sails.config.builtStaticContent.markdownPages,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath,
      pageTitleForMeta,
      pageDescriptionForMeta,
    };

  }


};

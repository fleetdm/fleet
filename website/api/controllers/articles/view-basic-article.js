module.exports = {

  friendlyName: 'View blog article',


  description: 'Display "Blog article" page.',

  urlWildcardSuffix: 'slug',

  inputs: {
    slug : {
      description: 'The relative path to the blog page from within this route.',
      example: 'blog/supported-browsers',
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
    let SECTION_URL_PREFIX = '/articles';

    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, {
      url: _.trimRight(SECTION_URL_PREFIX + '/' + slug)
    });
    let needsRedirectMaybe = (!thisPage);

    if (needsRedirectMaybe) {
      // Creating a lower case, repeating-slashless slug
      let multipleSlashesRegex = /\/{2,}/g;
      let modifiedslug = slug.toLowerCase().replace(multipleSlashesRegex, '/');
      // Finding the appropriate page content using the modified slug.
      let revisedPage = _.find(sails.config.builtStaticContent.markdownPages, {
        url: _.trimRight(SECTION_URL_PREFIX + '/' + _.trim(modifiedslug, '/'), '/')
      });
      if(revisedPage) {
        // If we matched a page with the modified slug, then redirect to that.
        throw {redirect: revisedPage.url};
      } else {
        // If no page was found, throw a 404 error.
        throw 'notFound';
      }
    }

    let pageTitleForMeta;
    if(!thisPage.title) {
      // If thisPage.title is 'Readme.md', we're on the docs landing page and we'll follow the title format of the other top level pages.
      pageTitleForMeta = 'Blog | Fleet for osquery';
    } else {
      // Otherwise we'll use the page title provided and format it accordingly.
      pageTitleForMeta = thisPage.title + ' | Fleet blog';
    }
    // Setting the meta description for this page if one was provided, otherwise setting a generic description.
    let pageDescriptionForMeta = thisPage.meta.description ? thisPage.meta.description : 'Fleet';
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

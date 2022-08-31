module.exports = {


  friendlyName: 'Build static content',


  description: 'Generate HTML partials from source files in fleetdm/fleet repo (e.g. docs in markdown, or queries in YAML), and configure metadata about the generated files so it is available in `sails.config.builtStaticContent`.',


  inputs: {
    articleFileName: {type: 'string', description: 'The filename of the article that will be converted into an HTML email partial', required: true},
  },


  fn: async function ({ dry, articleFileName }) {

    let path = require('path');
    let YAML = require('yaml');

    // FUTURE: If we ever need to gather source files from other places or branches, etc, see git history of this file circa 2021-05-19 for an example of a different strategy we might use to do that.
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    let APP_PATH_TO_COMPILED_EMAIL_PARTIALS = 'views/emails/newsletter-partials';

    let extensionedArticleFileName = articleFileName;
    // Delete existing HTML output from previous runs, if any.
    if(!_.endsWith(articleFileName, '.md')) {
      // If the file was specified without a file extension, we'll add `.md` to the provided filename.
      extensionedArticleFileName = extensionedArticleFileName + '.md';
      sails.log.warn('The filename provided is missing the .md file extension, appending `.md` to the provided articleFileName: '+extensionedArticleFileName)
    }
    let unextensionedArticleFileName = _.trimRight(extensionedArticleFileName, '.md');
    // Find the Markdown file in the articles folder
    let markdownFileToConvert = path.resolve(path.join(topLvlRepoPath, '/articles/'+extensionedArticleFileName));

    if(!markdownFileToConvert) { // If we couldn't find the file specified, throw an error
      throw new Error('Error: No article found with the filename: '+articleFileName);
    }

    if (path.extname(markdownFileToConvert) !== '.md') {// If this file doesn't end in `.md`: skip it (we won't create a page for it)
      throw new Error('Error: The specified file ('+articleFileName+') is not a valid Markdown file.'+markdownFileToConvert);
    }

    // Get the raw Markdown from the file.
    let mdStringForEmails = await sails.helpers.fs.read(markdownFileToConvert);


    let embeddedMetadata = {};
    for (let tag of (mdStringForEmails.match(/<meta[^>]*>/igm)||[])) {
      let name = tag.match(/name="([^">]+)"/i)[1];
      let value = tag.match(/value="([^">]+)"/i)[1];
      embeddedMetadata[name] = value;
    }//∞

    if(!embeddedMetadata.category) {
      throw new Error('Error: the Markdown article is missing a category meta tag. To resolve: add a category meta tag to the Markdown file');
    }

    if(!embeddedMetadata.articleTitle) {
      throw new Error('Error: the Markdown article is missing a articleTitle meta tag. To resolve: add an articleTitle meta tag to the Markdown file');
    }

    let extensionedFileNameForEmailPartial = embeddedMetadata.category+'-'+unextensionedArticleFileName.replace(/\./g, '-')+'.ejs';

    // Remove the meta tags from the final Markdown file before we convert it.
    mdStringForEmails = mdStringForEmails.replace(/<meta[^>]*>/igm, '');

    // Convert Markdown to HTML
    let htmlEmailString = await sails.helpers.strings.toHtmlEmail(mdStringForEmails);

    let pageRelSourcePath = path.relative(path.join(topLvlRepoPath, 'articles/'), path.resolve(markdownFileToConvert));

    let pageUnextensionedLowercasedRelPath = (
      pageRelSourcePath
      .replace(/(^|\/)([^/]+)\.[^/]*$/, '$1$2')
      .split(/\//).map((fileOrFolderName) => fileOrFolderName.toLowerCase()).join('/')
    );

    let htmlEmailOutputPath = path.resolve(sails.config.appPath, path.join(APP_PATH_TO_COMPILED_EMAIL_PARTIALS, extensionedFileNameForEmailPartial));

    // If an HTML partial exists for this article, we'll delete the old version and continue.
    if(path.resolve(htmlEmailOutputPath)) {
      sails.log.warn('Warning: An HTML partial for the Markdown article specified already exists. The old file will be replaced with the HTML partial generated.')
      await sails.helpers.fs.rmrf(htmlEmailOutputPath);
    }
    sails.log('Generated HTML partial for the Fleet Newsletter at: '+htmlEmailOutputPath);
    await sails.helpers.fs.write(htmlEmailOutputPath, htmlEmailString);


  }


};

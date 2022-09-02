module.exports = {


  friendlyName: 'Build HTML email',


  description: 'Generate an HTML partial for the Fleet newsletter from Markdown files in the articles/ folder of the fleetdm/fleet repo.',


  inputs: {
    articleFileName: {type: 'string', description: 'The filename of the article that will be converted into an HTML email partial', required: true},
  },


  fn: async function ({ articleFileName }) {

    let path = require('path');
    let YAML = require('yaml');

    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    let APP_PATH_TO_COMPILED_EMAIL_PARTIALS = 'views/emails/newsletter-partials';

    let extensionedArticleFileName = articleFileName;

    // Since this script only handles Markdown files in the articles/ folders, we'll make the file extension optional.
    if(!_.endsWith(articleFileName, '.md')) {
      // If the file was specified without a file extension, we'll add `.md` to the provided filename.
      extensionedArticleFileName = extensionedArticleFileName + '.md';
      sails.log.warn('The filename provided is missing the .md file extension, appending `.md` to the provided articleFileName: '+articleFileName)
    }

    let unextensionedArticleFileName = _.trimRight(extensionedArticleFileName, '.md');

    // Find the Markdown file in the articles folder
    let markdownFileToConvert = path.resolve(path.join(topLvlRepoPath, '/articles/'+extensionedArticleFileName));

    if(!markdownFileToConvert) { // If we couldn't find the file specified, throw an error
      throw new Error('Error: No Markdown file in found in the top level articles/ folder with the filename: '+articleFileName);
    }

    if (path.extname(markdownFileToConvert) !== '.md') {// If this file doesn't end in `.md`: skip it (we won't create a page for it)
      throw new Error('Error: The specified file ('+articleFileName+') is not a valid Markdown file.'+markdownFileToConvert);
    }

    // Get the raw Markdown from the file.
    let mdString = await sails.helpers.fs.read(markdownFileToConvert);

    // Get the relative path of the Markdown file we are converting
    let pageRelSourcePath = path.relative(path.join(topLvlRepoPath, 'articles/'), path.resolve(markdownFileToConvert));

    let embeddedMetadata = {};
    try {
      for (let tag of (mdString.match(/<meta[^>]*>/igm)||[])) {
        let name = tag.match(/name="([^">]+)"/i)[1];
        let value = tag.match(/value="([^">]+)"/i)[1];
        embeddedMetadata[name] = value;
      }//∞
    } catch(err) {
      throw new Error('An error occured while parsing <meta> tags in the Markdown file. Tip: Check the markdown file that is being converted to an email and make sure it doesn\'t contain any code snippets with <meta> inside, as this can fool the build script. Full error: '+err);
    }

    if(!embeddedMetadata.category) {
      throw new Error('Error: the Markdown article is missing a category meta tag. To resolve: add a category meta tag to the Markdown file');
    }

    let extensionedFileNameForEmailPartial = embeddedMetadata.category+'-'+unextensionedArticleFileName.replace(/\./g, '-')+'.ejs';

    // Remove the meta tags from the final Markdown file before we convert it.
    mdString = mdString.replace(/<meta[^>]*>/igm, '');

    // Find and remove any other HTML elements in the markdown file, note: this regex will match all of the content wrapped within an html element
    for (let htmlElement of (mdString.match(/<([A-Za-z\-]+[^\s])[\s\S]+?<\/\1>/igm) || [])) {
      sails.log.warn('Removing a HTML element from the Markdown file before converting it into an HTML email: \n',htmlElement)
      mdString = mdString.replace(htmlElement, '');
    }

    // Convert Markdown to HTML
    let htmlEmailString = await sails.helpers.strings.toHtmlEmail(mdString);

    // Replace relative links with links to fleetdm.com
    htmlEmailString = htmlEmailString.replace(/(href="(\.\/[^"]+|\.\.\/[^"]+)")/g, (hrefString)=>{// « Modify path-relative links like `./…` and `../…` to make them absolute.  (See https://github.com/fleetdm/fleet/issues/706#issuecomment-884641081 for more background)
      let oldRelPath = hrefString.match(/href="(\.\/[^"]+|\.\.\/[^"]+)"/)[1];

      let referencedPageSourcePath = path.resolve(path.join(topLvlRepoPath, 'articles/', pageRelSourcePath), '../', oldRelPath);

      let possibleReferencedUrlHash = oldRelPath.match(/(\.md#)([^/]*$)/) ? oldRelPath.match(/(\.md#)([^/]*$)/)[2] : false;

      let referencedPageNewUrl = 'https://fleetdm.com/' +
      (
        (path.relative(topLvlRepoPath, referencedPageSourcePath).replace(/(^|\/)([^/]+)\.[^/]*$/, '$1$2').split(/\//).map((fileOrFolderName) => fileOrFolderName.toLowerCase()).join('/'))
        .split(/\//)
        .map((fileOrFolderName) => encodeURIComponent(fileOrFolderName.replace(/^[0-9]+[\-]+/,''))).join('/')
      ).replace(/\/?readme\.?m?d?$/i, '');

      if(possibleReferencedUrlHash) {
        referencedPageNewUrl = referencedPageNewUrl + '#' + encodeURIComponent(possibleReferencedUrlHash);
      }
      return `href="${referencedPageNewUrl}"`;
    });

    // Find the location where this file will be saved.
    let htmlEmailOutputPath = path.resolve(sails.config.appPath, path.join(APP_PATH_TO_COMPILED_EMAIL_PARTIALS, extensionedFileNameForEmailPartial));

    // Delete existing HTML output from previous runs, if any exists.
    await sails.helpers.fs.rmrf(htmlEmailOutputPath);

    sails.log('Generated HTML partial from a Markdown article at: '+htmlEmailOutputPath);

    // Save the HTML output in website/pages/emails/newsletter-partials
    await sails.helpers.fs.write(htmlEmailOutputPath, htmlEmailString);
  }


};

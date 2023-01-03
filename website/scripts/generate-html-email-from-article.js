module.exports = {


  friendlyName: 'Generate HTML email from article',


  description: 'Generate an HTML partial for the Fleet newsletter from Markdown files in the articles/ folder of the fleetdm/fleet repo.',


  inputs: {
    articleFilename: {type: 'string', description: 'The filename of the article that will be converted into an HTML email partial', required: true},
  },


  fn: async function ({ articleFilename }) {

    let path = require('path');

    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    let APP_PATH_TO_COMPILED_EMAIL_PARTIALS = 'views/emails/newsletter';


    let extensionedArticleFilename = articleFilename;

    // Since this script only handles Markdown files in the articles/ folders, we'll make the file extension optional.
    if(!_.endsWith(articleFilename, '.md')) {
      // If the file was specified without a file extension, we'll add `.md` to the provided filename, and log a warning.
      extensionedArticleFilename = extensionedArticleFilename + '.md';
      sails.log.warn('The filename provided is missing the .md file extension, appending `.md` to the provided articleFilename: '+articleFilename);
    }

    // Get the filename without the .md file extension. This will be used to build the final filename
    let unextensionedArticleFilename = _.trimRight(extensionedArticleFilename, '.md');

    // Build the filename for the final HTML partial.
    let extensionedFileNameForEmailPartial = 'email-article-'+unextensionedArticleFilename.replace(/\./g, '-')+'.ejs';

    // Find the Markdown file in the articles folder
    let markdownFileToConvert = path.resolve(path.join(topLvlRepoPath, '/articles/'+extensionedArticleFilename));

    if(!markdownFileToConvert) { // If we couldn't find the file specified, throw an error
      throw new Error('Error: No Markdown file in found in the top level articles/ folder with the filename: '+articleFilename);
    }

    // If the file specified is not a markdown file, throw an error.
    if (path.extname(markdownFileToConvert) !== '.md') {
      throw new Error('Error: The specified file ('+articleFilename+') is not a valid Markdown file.'+markdownFileToConvert);
    }

    // Get the raw Markdown from the file.
    let mdString = await sails.helpers.fs.read(markdownFileToConvert);

    // Get the relative path of the Markdown file we are converting
    let pageRelSourcePath = path.relative(path.join(topLvlRepoPath, 'articles/'), path.resolve(markdownFileToConvert));

    // Remove any meta tags from the Markdown file before we convert it.
    mdString = mdString.replace(/<meta[^>]*>/igm, '');

    // Find and remove any iframe elements in the markdown file
    for (let matchedIframe of (mdString.match(/<(iframe)[\s\S]+?<\/iframe>/igm) || [])) {
      sails.log.warn('Removing an <iframe> element from the Markdown file before converting it into an HTML email: \n',matchedIframe);
      mdString = mdString.replace(matchedIframe, '');
    }

    // Convert Markdown to HTML
    let htmlEmailString = await sails.helpers.strings.toHtmlEmail(mdString);

    // Modify path-relative links in the final HTML like `./…` and `../…` to make them absolute.  (See https://github.com/fleetdm/fleet/issues/706#issuecomment-884641081 for more background)
    htmlEmailString = htmlEmailString.replace(/(href="(\.\/[^"]+|\.\.\/[^"]+)")/g, (hrefString)=>{
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

module.exports = {


  friendlyName: 'Generate newsletter emails',


  description: 'Generate HTML partials for the Fleet newsletter from any Markdown article in the articles/ folder that has a "newsletters" category and does not yet have a generated email partial.',


  inputs: {
    regenerate: {
      type: 'boolean',
      description: 'If provided, the existing email partials created for newsletter articles will be regenerated.',
      defaultsTo: false,
    }
  },

  fn: async function ({regenerate}) {

    let path = require('path');

    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    let APP_PATH_TO_COMPILED_EMAIL_PARTIALS = 'views/emails/newsletter';

    let ARTICLES_FOLDER_PATH = path.join(topLvlRepoPath, 'articles');

    // Get a list of every Markdown file in the top level articles/ folder.
    let articleFilenames = await sails.helpers.fs.ls.with({
      dir: ARTICLES_FOLDER_PATH,
      depth: 1,
      includeDirs: false,
      includeSymlinks: false,
    });

    for (let markdownFileToConvert of articleFilenames) {

      // Only handle Markdown files.
      if (path.extname(markdownFileToConvert) !== '.md') {
        continue;
      }

      // Get the raw Markdown from the file.
      let mdString = await sails.helpers.fs.read(markdownFileToConvert);

      // Skip any article that isn't in the "newsletters" category.
      // (Newsletter articles have a `<meta name="category" value="newsletters">` tag.)
      if (!mdString.match(/<meta[^>]*name="category"[^>]*value="newsletters"[^>]*>/i)) {
        continue;
      }

      // Get the filename without the .md file extension. This will be used to build the final filename.
      let unextensionedArticleFilename = _.trimRight(path.basename(markdownFileToConvert), '.md');

      // Build the filename for the final HTML partial.
      let extensionedFileNameForEmailPartial = 'email-article-'+unextensionedArticleFilename.replace(/\./g, '-')+'.ejs';

      // Find the location where this file will be saved.
      let htmlEmailOutputPath = path.resolve(sails.config.appPath, path.join(APP_PATH_TO_COMPILED_EMAIL_PARTIALS, extensionedFileNameForEmailPartial));

      // // If an email partial has already been generated for this article, skip it.
      if (!regenerate && await sails.helpers.fs.exists(htmlEmailOutputPath)) {
        continue;
      }

      // Get the relative path of the Markdown file we are converting
      let pageRelSourcePath = path.relative(ARTICLES_FOLDER_PATH, path.resolve(markdownFileToConvert));

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

      sails.log('Generated HTML partial from a Markdown article at: '+htmlEmailOutputPath);

      // Save the HTML output in website/views/emails/newsletter
      await sails.helpers.fs.write(htmlEmailOutputPath, htmlEmailString, regenerate);
    }
  }


};

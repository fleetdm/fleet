module.exports = {


  friendlyName: 'Compile markdown content',


  // TODO: Make this explanation better or refactor, because actually this does a lot more than just that, including cloning the source git repo
  description: 'Compile documentation templates from markdown.',
  // Also, FUTURE: dissect some of the code from here https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L16-L22
  // and use those building blocks directly instead of depending on doctemplater later and thus unnecessarily duplicating work.  Also the other related code in sailsjs docs mentioned in https://github.com/fleetdm/fleet/issues/706
  // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
  // Clone repo
  // let topLvlCachePath = path.resolve(sails.config.paths.tmp, `built-static-content/`);
  // await sails.helpers.fs.rmrf(topLvlCachePath);
  // let repoCachePath = path.join(topLvlCachePath, `cloned-repo-${Date.now()}-${Math.round(Math.random()*100)}`);
  // await sails.helpers.process.executeCommand(`git clone git://github.com/fleetdm/fleet.git ${repoCachePath}`);
  // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -


  inputs: {

    repoPath: {
      description: 'The path of the subdirectory in the Git repo to compile from.',
      type: 'string',
      example: 'docs/',
      required: true
    },

    repoBranch: {
      description: 'The name of the branch to compile from.',
      type: 'string',
      defaultsTo: 'master'
    },

    repoUrl: {
      description: 'The git:// URL of the remote Git repo to compile from.',
      type: 'string',
      defaultsTo: 'git://github.com/fleetdm/fleet.git',
    }

  },


  exits: {
    success: { outputDescription: 'A list of metadata about all generated HTML files.', outputType: [{}] }
  },


  fn: async function ({repoPath, repoBranch, repoUrl}) {

    let path = require('path');
    let cheerio = require('cheerio');
    let DocTemplater = {};// require('doc-templater');
    if (true) {
      throw new Error('This helper has been retired.  TODO: delete it.');
    }

    sails.log.info('Compiling `%s` docs from the `%s` branch of `%s`...', repoPath, repoBranch, repoUrl);

    // Relative path within this app where output will be written.
    let htmlOutputPath = path.join('views/partials/built-from-markdown/', _.kebabCase(repoPath));

    // Relative path within this app where temporary menu data will be written.
    // (This is consumed later in this file.)
    let jsMenuOutputPath = path.join('.tmp/doc-templater/menus/', `${_.kebabCase(repoPath)}.tmp-menu.json`);// fka "the .jsmenu file"

    // Delete existing output from previous runs, if any.
    await sails.helpers.fs.rmrf(path.resolve(sails.config.appPath, htmlOutputPath));
    await sails.helpers.fs.rmrf(path.resolve(sails.config.appPath, jsMenuOutputPath));

    // Compile the markdown into HTML files and a JSON file (aka "jsmenu") representing
    // the manifest of all compiled HTML files and their hierarchy.
    await new Promise((resolve, reject)=>{
      DocTemplater().build([{
        remote: repoUrl,
        branch: repoBranch,
        remoteSubPath: repoPath,
        outputExtension: 'ejs',//« the file extension for resulting HTML files
        htmlDirPath: htmlOutputPath,
        jsMenuPath: jsMenuOutputPath,
        beforeConvert: (mdString, proceed)=>{// This function is applied to each template before the markdown is converted to markup
          // Based on the github-flavored markdown's language annotation, (e.g. ```js```) add a temporary marker to code blocks that can be parsed post-md-compilation by the `afterConvert()` lifecycle hook
          // Note: This is an HTML comment because it is easy to over-match and "accidentally" add it underneath each code block as well (being an HTML comment ensures it doesn't show up or break anything)
          let LANG_MARKER_PREFIX = '<!-- __LANG=%';
          let LANG_MARKER_SUFFIX = '%__ -->';
          let modifiedMd = mdString.replace(/(```)([a-zA-Z0-9\-]*)(\s*\n)/g, '$1\n' + LANG_MARKER_PREFIX + '$2' + LANG_MARKER_SUFFIX + '$3');

          // FUTURE: implement a way of skipping non-markdown files, such as pngs (maybe here, but prbly just make it built-in)

          return proceed(undefined, modifiedMd);
        },
        afterConvert: (html, proceed)=>{// This function is applied to each template after the markdown is converted to markup

          let modifiedHtml = html;

          // Replace github emoji with unicode emojis
          // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
          // TODO: actually sub unicode, instead of the following  (there's probably an open source lib out there to do it)
          // modifiedHtml = html.replace(/\:white_check_mark\:/g, '<i class="sails-icon icon-plus"></i>');
          // modifiedHtml = modifiedHtml.replace(/\:white_large_square\:/g, '<i class="sails-icon icon-minus"></i>');
          // modifiedHtml = modifiedHtml.replace(/\:heavy_multiplication_x\:/g, '<i class="sails-icon icon-times"></i>');
          // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

          // TODO: As equivalent concepts are identified in the Fleet docs (e.g. in the API reference), maybe bring this back:
          // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
          // // Replace ((bubble))s with HTML
          // modifiedHtml = modifiedHtml.replace(/\(\(([^())]*)\)\)/g, '<bubble type="$1" class="colors"><span is="bubble-heart"></span></bubble>');
          // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

          // Flag <h2>, <h3>, <h4>, and <h5> tags with the `permalinkable` directive, so they can be clicked
          // e.g. ?q=transport-compatibility
          let $ = cheerio.load(modifiedHtml);
          $('h2, h3, h4, h5').each(function() {
            let content = $(this).text() || '';

            // build the URL slug suffix
            let slug = content
              .replace(/[\?\!\.\-\_\:\;\'\"]/g, '') // punctuation => gone
              .replace(/\s/g, '-') // spaces => dashes
              .toLowerCase();

            // set the "permalink" HTML attr to the slug
            $(this).attr('permalink', slug);

            if ($(this) && typeof $(this).wrap === 'function') {// this was throwing ".wrap is undefined"
              $(this).wrap('<div class="permalink-header"></div>');
            }

          });
          modifiedHtml = $.html();

          // Modify links
          modifiedHtml = modifiedHtml.replace(/(href="https?:\/\/([^"]+)")/g, (hrefString)=>{
            // Check if this is an external link (like https://google.com) but that is ALSO not a link
            // to some page on the destination site where this will be hosted, like `(*.)?sailsjs.com`.
            // If external, add target="_blank" so the link will open in a new tab.
            let isExternal = ! hrefString.match(/^href=\"https?:\/\/([^\.]+\.)*fleetdm\.com/g);
            if (isExternal) {
              return hrefString.replace(/(href="https?:\/\/([^"]+)")/g, '$1 target="_blank"');
            } else {
              // Otherwise, change the link to be web root relative.
              // (e.g. 'href="http://sailsjs.com/documentation/concepts"'' becomes simply 'href="/documentation/concepts"'')
              // Note: See the Git version history of this file for examples of ways this can work across versioned subdomains.
              return hrefString.replace(/href="https?:\/\//, '').replace(/^fleetdm\.com/, 'href="');
            }
          });//∞

          // Add the appropriate class to the `<code>` based on the temporary marker that was added in the `beforeConvert` function above
          // console.log('RAN AFTER HOOK, found: ',modifiedHtml.match(/(<code)([^>]*)(>\s*)(\&lt;!--\s*__LANG=\%[^\%]*\%__\s*--\&gt;)/g));
          modifiedHtml = modifiedHtml.replace(// Interpret `js` as `javascript`
            // $1     $2     $3   $4
            /(<code)([^>]*)(>\s*)(\&lt;!-- __LANG=\%js\%__ --\&gt;)\s*/gm,
            '$1 class="javascript"$2$3'
          );
          modifiedHtml = modifiedHtml.replace(// Interpret `sh` and `bash` as `bash`
            // $1     $2     $3   $4
            /(<code)([^>]*)(>\s*)(\&lt;!-- __LANG=\%(bash|sh)\%__ --\&gt;)\s*/gm,
            '$1 class="bash"$2$3'
          );
          modifiedHtml = modifiedHtml.replace(// When unspecified, default to `text`
            // $1     $2     $3   $4
            /(<code)([^>]*)(>\s*)(\&lt;!-- __LANG=\%\%__ --\&gt;)\s*/gm,
            '$1 class="nohighlight"$2$3'
          );
          modifiedHtml = modifiedHtml.replace(// Finally, nab the rest, leaving the code language as-is.
            // $1     $2     $3   $4               $5    $6
            /(<code)([^>]*)(>\s*)(\&lt;!-- __LANG=\%)([^%]+)(\%__ --\&gt;)\s*/gm,
            '$1 class="$5"$2$3'
          );

          return proceed(undefined, modifiedHtml);
        },
      }], (err)=>{
        if (err) {
          reject(err);
        } else {
          resolve();
        }
      });//_∏_
    });

    // Now return the menu data, to provide a nicer way of programmatically working with it.
    // But first, we'll clean it up a bit.
    let filesGenerated = await sails.helpers.fs.readJson(jsMenuOutputPath);

    // TODO: bring in the other cleanup in marshal-doc-page-metadata from sailsjs.com repo

    // Since the format from the doc-templater package can be a little bit misleading,
    // we'll also munge the resulting data a little bit.
    for (let fileInfo of filesGenerated) {

      fileInfo.path = fileInfo.fullPathAndFileName;// « for clarity (it's not technically the full path)
      delete fileInfo.fullPathAndFileName;

      fileInfo.fallbackTitle = sails.helpers.strings.toSentenceCase(path.basename(fileInfo.templateTitle, '.ejs'));// « for clarity (the page isn't a template, necessarily, and this title is just a guess.  Display title will, more likely than not, come from a <docmeta> tag -- see the bottom of the original, raw unformatted markdown of any page in the sailsjs docs for an example of how to use docmeta tags)
      delete fileInfo.templateTitle;

      delete fileInfo.data.lastModified;// « for clarity (this isn't the timestamp you're expecting, so we delete it)
    }//∞

    return filesGenerated;

  }


};

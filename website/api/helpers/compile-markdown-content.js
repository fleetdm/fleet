module.exports = {


  friendlyName: 'Compile markdown docs',


  description: 'Compile documentation templates from markdown.',


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


  fn: async function ({repoPath, repoBranch, repoUrl}) {

    let path = require('path');
    let cheerio = require('cheerio');
    let DocTemplater = require('doc-templater');

    sails.log.info('Compiling `%s` docs from the `%s` branch of `%s`...', repoPath, repoBranch, repoUrl);

    // Relative paths within this app where output will be written.
    let htmlOutputPath = path.join('views/partials/compiled-from-markdown/', _.kebabCase(repoPath));
    let jsMenuOutputPath = path.join('.tmp/doc-templater/menus/', `${_.kebabCase(repoPath)}.menu.json`);// fka "the .jsmenu file"

    // Delete existing output from previous runs, if any.
    await sails.helpers.fs.rmrf(path.resolve(sails.config.appPath, htmlOutputPath));
    await sails.helpers.fs.rmrf(path.resolve(sails.config.appPath, jsMenuOutputPath));

    // Compile the markdown into HTML templates and menu files.
    // (Note that "lastModified" in the resulting jsmenu files can be misleading)
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

  }


};

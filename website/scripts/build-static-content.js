module.exports = {


  friendlyName: 'Build static content',


  description: 'Generate HTML partials from source files in fleetdm/fleet repo (e.g. docs in markdown, or queries in YAML), and configure metadata about the generated files so it is available in `sails.config.builtStaticContent`.',


  inputs: {
    dry: { type: 'boolean', description: 'Whether to make this a dry run.  (.sailsrc file will not be overwritten.  HTML files will not be generated.)' },
  },


  fn: async function ({ dry }) {

    let path = require('path');
    let YAML = require('yaml');

    // FUTURE: If we ever need to gather source files from other places or branches, etc, see git history of this file circa 2021-05-19 for an example of a different strategy we might use to do that.
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    // The data we're compiling will get built into this dictionary and then written on top of the .sailsrc file.
    let builtStaticContent = {};

    await sails.helpers.flow.simultaneously([
      async()=>{// Parse query library from YAML and prepare to bake them into the Sails app's configuration.
        let RELATIVE_PATH_TO_QUERY_LIBRARY_YML_IN_FLEET_REPO = 'docs/01-Using-Fleet/standard-query-library/standard-query-library.yml';
        let yaml = await sails.helpers.fs.read(path.join(topLvlRepoPath, RELATIVE_PATH_TO_QUERY_LIBRARY_YML_IN_FLEET_REPO)).intercept('doesNotExist', (err)=>new Error(`Could not find standard query library YAML file at "${RELATIVE_PATH_TO_QUERY_LIBRARY_YML_IN_FLEET_REPO}".  Was it accidentally moved?  Raw error: `+err.message));

        let queriesWithProblematicResolutions = [];
        let queriesWithProblematicContributors = [];
        let queriesWithProblematicTags = [];
        let queries = YAML.parseAllDocuments(yaml).map((yamlDocument)=>{
          let query = yamlDocument.toJSON().spec;
          query.kind = yamlDocument.toJSON().kind;
          query.slug = _.kebabCase(query.name);// « unique slug to use for routing to this query's detail page
          if ((query.resolution !== undefined && !_.isString(query.resolution)) || (query.kind !== 'policy' && _.isString(query.resolution))) {
            // console.log(typeof query.resolution);
            queriesWithProblematicResolutions.push(query);
          } else if (query.resolution === undefined) {
            query.resolution = 'N/A';// « We set this to a string here so that the data type is always string.  We use N/A so folks can see there's no remediation and contribute if desired.
          }
          if (query.tags) {
            if(!_.isString(query.tags)) {
              queriesWithProblematicTags.push(query);
            } else {
              // Splitting tags into an array to format them.
              let tagsToFormat = query.tags.split(',');
              let formattedTags = [];
              for (let tag of tagsToFormat) {
                if(tag !== '') {// « Ignoring any blank tags caused by trailing commas in the YAML.
                  // Removing any extra whitespace from tags and changing them to be in lower case.
                  formattedTags.push(_.trim(tag.toLowerCase()));
                }
              }
              // Removing any duplicate tags.
              query.tags = _.uniq(formattedTags);
            }
          } else {
            query.tags = []; // « if there are no tags, we set query.tags to an empty array so it is always the same data type.
          }

          // GitHub usernames may only contain alphanumeric characters or single hyphens, and cannot begin or end with a hyphen.
          if (!query.contributors || (query.contributors !== undefined && !_.isString(query.contributors)) || query.contributors.split(',').some((contributor) => contributor.match('^[^A-za-z0-9].*|[^A-Za-z0-9-]|.*[^A-za-z0-9]$'))) {
            queriesWithProblematicContributors.push(query);
          }

          return query;
        });
        // Report any errors that were detected along the way in one fell swoop to avoid endless resubmitting of PRs.
        if (queriesWithProblematicResolutions.length >= 1) {
          throw new Error('Failed parsing YAML for query library: The "resolution" of a query should either be absent (undefined) or a single string (not a list of strings).  And "resolution" should only be present when a query\'s kind is "policy".  But one or more queries have an invalid "resolution": ' + _.pluck(queriesWithProblematicResolutions, 'slug').sort());
        }//•
        if (queriesWithProblematicTags.length >= 1) {
          throw new Error('Failed parsing YAML for query library: The "tags" of a query should either be absent (undefined) or a single string (not a list of strings). "tags" should be be be seperated by a comma.  But one or more queries have invalid "tags": ' + _.pluck(queriesWithProblematicTags, 'slug').sort());
        }
        // Assert uniqueness of slugs.
        if (queries.length !== _.uniq(_.pluck(queries, 'slug')).length) {
          throw new Error('Failed parsing YAML for query library: Queries as currently named would result in colliding (duplicate) slugs.  To resolve, rename the queries whose names are too similar.  Note the duplicates: ' + _.pluck(queries, 'slug').sort());
        }//•
        // Report any errors that were detected along the way in one fell swoop to avoid endless resubmitting of PRs.
        if (queriesWithProblematicContributors.length >= 1) {
          throw new Error('Failed parsing YAML for query library: The "contributors" of a query should be a single string of valid GitHub user names (e.g. "zwass", or "zwass,noahtalerman,mikermcneil").  But one or more queries have an invalid "contributors" value: ' + _.pluck(queriesWithProblematicContributors, 'slug').sort());
        }//•

        // Get a distinct list of all GitHub usernames from all of our queries.
        // Map all queries to build a list of unique contributor names then build a dictionary of user profile information from the GitHub Users API
        const githubUsernames = queries.reduce((list, query) => {
          if (!queriesWithProblematicContributors.find((element) => element.slug === query.slug)) {
            list = _.union(list, query.contributors.split(','));
          }
          return list;
        }, []);

        // Talk to GitHub and get additional information about each contributor.
        let githubDataByUsername = {};
        await sails.helpers.flow.simultaneouslyForEach(githubUsernames, async(username)=>{
          githubDataByUsername[username] = await sails.helpers.http.get.with({
            url: 'https://api.github.com/users/' + encodeURIComponent(username),
            headers: { 'User-Agent': 'Fleet-Standard-Query-Library', Accept: 'application/vnd.github.v3+json' }
          });
        });//∞

        // Now expand queries with relevant profile data for the contributors.
        for (let query of queries) {
          let usernames = query.contributors.split(',');
          let contributorProfiles = [];
          for (let username of usernames) {
            contributorProfiles.push({
              name: githubDataByUsername[username].name,
              handle: githubDataByUsername[username].login,
              avatarUrl: githubDataByUsername[username].avatar_url,
              htmlUrl: githubDataByUsername[username].html_url,
            });
          }
          query.contributors = contributorProfiles;
        }

        // Attach to what will become configuration for the Sails app.
        builtStaticContent.queries = queries;
        builtStaticContent.queryLibraryYmlRepoPath = RELATIVE_PATH_TO_QUERY_LIBRARY_YML_IN_FLEET_REPO;
      },
      async()=>{// Parse markdown pages, compile & generate HTML files, and prepare to bake directory trees into the Sails app's configuration.
        let APP_PATH_TO_COMPILED_PAGE_PARTIALS = 'views/partials/built-from-markdown';

        // Delete existing HTML output from previous runs, if any.
        await sails.helpers.fs.rmrf(path.resolve(sails.config.appPath, APP_PATH_TO_COMPILED_PAGE_PARTIALS));

        builtStaticContent.markdownPages = [];// « dir tree representation that will be injected into Sails app's configuration

        let SECTION_INFOS_BY_SECTION_REPO_PATHS = {
          'docs/':     { urlPrefix: '/docs', },
          'handbook/': { urlPrefix: '/handbook', },
          'articles/': { urlPrefix: '/articles', }
        };
        let rootRelativeUrlPathsSeen = [];
        for (let sectionRepoPath of Object.keys(SECTION_INFOS_BY_SECTION_REPO_PATHS)) {// FUTURE: run this in parallel
          let thinTree = await sails.helpers.fs.ls.with({
            dir: path.join(topLvlRepoPath, sectionRepoPath),
            depth: 100,
            includeDirs: false,
            includeSymlinks: false,
          });

          for (let pageSourcePath of thinTree) {// FUTURE: run this in parallel

            // Crunch some paths (used for determining the URL, etc below.)
            // > Inspired by https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L308-L313
            // > And https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L107-L132
            let pageRelSourcePath = path.relative(path.join(topLvlRepoPath, sectionRepoPath), path.resolve(pageSourcePath));
            let pageUnextensionedLowercasedRelPath = (
              pageRelSourcePath
              .replace(/(^|\/)([^/]+)\.[^/]*$/, '$1$2')
              .split(/\//).map((fileOrFolderName) => fileOrFolderName.toLowerCase()).join('/')
            );
            let RX_README_FILENAME = /\/?readme\.?m?d?$/i;// « for matching `readme` or `readme.md` (case-insensitive) at the end of a file path

            // Determine this page's default (fallback) display title.
            // (README pages use their folder name as their fallback title.)
            let fallbackPageTitle;
            if (pageSourcePath.match(RX_README_FILENAME)) {
              // console.log(pageRelSourcePath.split(/\//).slice(-2)[0], path.basename(pageRelSourcePath), pageRelSourcePath);
              fallbackPageTitle = sails.helpers.strings.toSentenceCase(pageRelSourcePath.split(/\//).slice(-2)[0]);
            } else {
              fallbackPageTitle = sails.helpers.strings.toSentenceCase(path.basename(pageSourcePath, path.extname(pageSourcePath)));
            }

            // Determine URL for this page
            let rootRelativeUrlPath = (
              (
                SECTION_INFOS_BY_SECTION_REPO_PATHS[sectionRepoPath].urlPrefix +
                '/' + (
                  pageUnextensionedLowercasedRelPath
                  .split(/\//).map((fileOrFolderName) => encodeURIComponent(fileOrFolderName.replace(/^[0-9]+[\-]+/,''))).join('/')// « Get URL-friendly by encoding characters and stripping off ordering prefixes (like the "1-" in "1-Using-Fleet") for all folder and file names in the path.
                )
              ).replace(RX_README_FILENAME, '')// « Interpret README files as special and map it to the URL representing its containing folder.
            );

            // Assert uniqueness of URL paths.
            if (rootRelativeUrlPathsSeen.includes(rootRelativeUrlPath)) {
              throw new Error('Failed compiling markdown content: Files as currently named would result in colliding (duplicate) URLs for the website.  To resolve, rename the pages whose names are too similar.  Duplicate detected: ' + rootRelativeUrlPath);
            }//•
            rootRelativeUrlPathsSeen.push(rootRelativeUrlPath);

            if (path.extname(pageSourcePath) !== '.md') {// If this file doesn't end in `.md`: skip it (we won't create a page for it)
              // > Inspired by https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L275-L276
              sails.log.verbose(`Skipping ${pageSourcePath}`);
            } else {// Otherwise, this is markdown, so: Compile to HTML, parse docpage metadata, and build+track it as a page
              sails.log.verbose(`Building page ${rootRelativeUrlPath} (from ${pageSourcePath})`);

              // Compile markdown to HTML.
              // > This includes build-time enablement of:
              // >  • syntax highlighting
              // >  • data type bubbles
              // >  • transforming relative markdown links to their fleetdm.com equivalents
              // >
              // > For more info about how these additional features work, see: https://github.com/fleetdm/fleet/issues/706#issuecomment-884622252
              // >
              // > • What about images referenced in markdown files? :: They need to be referenced using an absolute URL src-- e.g. ![](https://fleetdm.com/images/foo.png)   See also https://github.com/fleetdm/fleet/issues/706#issuecomment-884641081 for reasoning.
              // > • What about GitHub-style emojis like `:white_check_mark:`?  :: Use actual unicode emojis instead.  Need to revisit this?  Visit https://github.com/fleetdm/fleet/pull/1380/commits/19a6e5ffc70bf41569293db44100e976f3e2bda7 for more info.
              let mdString = await sails.helpers.fs.read(pageSourcePath);
              mdString = mdString.replace(/(```)([a-zA-Z0-9\-]*)(\s*\n)/g, '$1\n' + '<!-- __LANG=%' + '$2' + '%__ -->' + '$3'); // « Based on the github-flavored markdown's language annotation, (e.g. ```js```) add a temporary marker to code blocks that can be parsed post-md-compilation when this is HTML.  Note: This is an HTML comment because it is easy to over-match and "accidentally" add it underneath each code block as well (being an HTML comment ensures it doesn't show up or break anything).  For more information, see https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L198-L202
              mdString = mdString.replace(/(<call-to-action[\s\S]+[^>\n+])\n+(>)/g, '$1$2'); // « Removes any newlines that might exist before the closing `>` when the <call-to-action> compontent is added to markdown files.
              let htmlString = await sails.helpers.strings.toHtml(mdString);
              htmlString = (// « Add the appropriate class to the `<code>` based on the temporary "LANG" markers that were just added above
                htmlString
                .replace(// Interpret `js` as `javascript`
                  // $1     $2     $3   $4
                  /(<code)([^>]*)(>\s*)(\&lt;!-- __LANG=\%js\%__ --\&gt;)\s*/gm,
                  '$1 class="javascript"$2$3'
                )
                .replace(// Interpret `sh` and `bash` as `bash`
                  // $1     $2     $3   $4
                  /(<code)([^>]*)(>\s*)(\&lt;!-- __LANG=\%(bash|sh)\%__ --\&gt;)\s*/gm,
                  '$1 class="bash"$2$3'
                )
                .replace(// When unspecified, default to `text`
                  // $1     $2     $3   $4
                  /(<code)([^>]*)(>\s*)(\&lt;!-- __LANG=\%\%__ --\&gt;)\s*/gm,
                  '$1 class="nohighlight"$2$3'
                )
                .replace(// Finally, nab the rest, leaving the code language as-is.
                  // $1     $2     $3   $4               $5    $6
                  /(<code)([^>]*)(>\s*)(\&lt;!-- __LANG=\%)([^%]+)(\%__ --\&gt;)\s*/gm,
                  '$1 class="$5"$2$3'
                )
              );
              // Throw an error if the compiled Markdown contains nested codeblocks (nested codeblocks meaning 3 backtick codeblocks nested inside a 4 backtick codeblock, or vice versa). Note: We're checking this after the markdown has been compiled because backticks (`) within codeblocks will be replaced with HTML entities (&#96;) and nested triple backticks can be easy to overmatch.
              if(htmlString.match(/(&#96;){3,4}[\s\S]+(&#96;){3}/g)){
                throw new Error('The compiled markdown has a codeblock (\`\`\`) nested inside of another codeblock (\`\`\`\`) at '+pageSourcePath+'. To resolve this error, remove the codeblock nested inside another codeblock from this file.');
              }
              htmlString = htmlString.replace(/\(\(([^())]*)\)\)/g, '<bubble type="$1" class="colors"><span is="bubble-heart"></span></bubble>');// « Replace ((bubble))s with HTML. For more background, see https://github.com/fleetdm/fleet/issues/706#issuecomment-884622252
              htmlString = htmlString.replace(/(href="(\.\/[^"]+|\.\.\/[^"]+)")/g, (hrefString)=>{// « Modify path-relative links like `./…` and `../…` to make them absolute.  (See https://github.com/fleetdm/fleet/issues/706#issuecomment-884641081 for more background)
                let oldRelPath = hrefString.match(/href="(\.\/[^"]+|\.\.\/[^"]+)"/)[1];
                // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
                // Note: This approach won't work as far as linking between handbook and docs
                // FUTURE: improve it so that it does.  This may involve pulling out URL determination as a separate, first step, then looking up the appropriate URL.
                // Currently this is a kinda duplicative hack, that just determines the appropriate URL in a similar way to all the code above...
                // -mikermcneil 2021-07-27
                // ```
                let referencedPageSourcePath = path.resolve(path.join(topLvlRepoPath, sectionRepoPath, pageRelSourcePath), '../', oldRelPath);
                let possibleReferencedUrlHash = oldRelPath.match(/(\.md#)([^/]*$)/) ? oldRelPath.match(/(\.md#)([^/]*$)/)[2] : false;
                let referencedPageNewUrl = 'https://fleetdm.com/' + (
                  (path.relative(topLvlRepoPath, referencedPageSourcePath).replace(/(^|\/)([^/]+)\.[^/]*$/, '$1$2').split(/\//).map((fileOrFolderName) => fileOrFolderName.toLowerCase()).join('/'))
                  .split(/\//).map((fileOrFolderName) => encodeURIComponent(fileOrFolderName.replace(/^[0-9]+[\-]+/,''))).join('/')
                ).replace(RX_README_FILENAME, '');
                if(possibleReferencedUrlHash) {
                  referencedPageNewUrl = referencedPageNewUrl + '#' + encodeURIComponent(possibleReferencedUrlHash);
                }
                // console.log(pageRelSourcePath, '»»  '+hrefString+' »»»»    href="'+referencedPageNewUrl+'"');
                // ```
                // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
                return `href="${referencedPageNewUrl}"`;
              });
              htmlString = htmlString.replace(/(href="https?:\/\/([^"]+)")/g, (hrefString)=>{// « Modify links that are potentially external
                // Check if this is an external link (like https://google.com) but that is ALSO not a link
                // to some page on the destination site where this will be hosted, like `(*.)?fleetdm.com`.
                // If external, add target="_blank" so the link will open in a new tab.
                // Note: links to blog.fleetdm.com will be treated as an external link.
                let isExternal = ! hrefString.match(/^href=\"https?:\/\/([^\.|blog]+\.)*fleetdm\.com/g);// « FUTURE: make this smarter with sails.config.baseUrl + _.escapeRegExp()
                // Check if this link is to fleetdm.com or www.fleetdm.com.
                let isBaseUrl = hrefString.match(/^(href="https?:\/\/)([^\.]+\.)*fleetdm\.com"$/g);
                if (isExternal) {

                  return hrefString.replace(/(href="https?:\/\/([^"]+)")/g, '$1 target="_blank"');
                } else {
                  // Otherwise, change the link to be web root relative.
                  // (e.g. 'href="http://sailsjs.com/documentation/concepts"'' becomes simply 'href="/documentation/concepts"'')
                  // > Note: See the Git version history of "compile-markdown-content.js" in the sailsjs.com website repo for examples of ways this can work across versioned subdomains.
                  if (isBaseUrl) {
                    return hrefString.replace(/href="https?:\/\//, '').replace(/([^\.]+\.)*fleetdm\.com/, 'href="/');
                  } else {
                    return hrefString.replace(/href="https?:\/\//, '').replace(/^fleetdm\.com/, 'href="');
                  }
                }

              });//∞

              // Modify images in the /articles folder
              if (sectionRepoPath === 'articles/') {
                // modifying relative links e.g. `../website/assets/images/articles/foo-200x300@2x.png`
                htmlString = htmlString.replace(/((?<=(src))="(\.\/[^"]+|\.\.\/[^"]+)?")/g, (srcString)=>{
                  let oldRelPath = srcString.match(/="(\.\/[^"]+|\.\.\/[^"]+)"/)[1];
                  let referencedPageSourcePath = path.resolve(path.join(topLvlRepoPath, sectionRepoPath, pageRelSourcePath), '../', oldRelPath);
                  // If the relative link goes to the image is in the website's assets folder (`website/assets/`) we'll modify the relative link
                  // to work on fleetdm.com e.g. ('../website/assets/images/articles/foo-300x900@2x.png' -> '/images/articles/foo-200x300@2x.png')
                  let isWebsiteAsset = referencedPageSourcePath.match(/(?<=\/website\/assets)(\/images\/(.+))/g)[0];
                  if(isWebsiteAsset) {
                    return '="'+isWebsiteAsset+'"';
                  } else {
                    // If the relative link doesn't go to the `website/assets/` folder, we'll throw an error.
                    throw new Error(`Failed compiling markdown content: An article page has an invalid image link ${srcString} at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, ensure this image has been added to 'website/assets/images/articles/' and link to it using a relative link e.g. '../website/assets/images/articles/foo-200x300@2x.png' OR a link to the image on fleetdm.com e.g. 'https://fleetdm.com/images/articles/foo-200x300@2x.png`);
                  }
                });//∞

                // Modify links to images hosted on fleetdm.com to link directly to the file in the `website/assets/` folder
                htmlString = htmlString.replace(/((?<=(src))="https?:\/\/([^"]+)")/g, (srcString)=>{
                  let isExternal = ! srcString.match(/=\"https?:\/\/([^\.|blog]+\.)*fleetdm\.com/g);
                  if (!isExternal) {
                    return srcString.replace(/=\"https?:\/\//, '').replace(/^fleetdm\.com/, '="');
                  } else {
                    return srcString;
                  }
                });//∞

              }

              // Find all H2s in handbook pages for the generated handbook index
              let linksForHandbookIndex = [];
              if(sectionRepoPath === 'handbook/') {
                for (let link of (mdString.match(/(\n\#\#\s.+)\n/g, '$1')||[])) {
                  let sectionInHandbookPage =  {};
                  // Remove any preceeding #s and any trailing newlines from the matched link
                  sectionInHandbookPage.headingText = link.replace(/\n## /, '').replace(/\n/g, '');
                  // Build the relative hash link for the matched heading
                  sectionInHandbookPage.hashLink = rootRelativeUrlPath+'#'+_.kebabCase(sectionInHandbookPage.headingText);
                  linksForHandbookIndex.push(sectionInHandbookPage);
                }
              }

              // Extract metadata from markdown.
              // > • Parsing meta tags (consider renaming them to just <meta>- or by now there's probably a more standard way of embedding semantics in markdown files; prefer to use that): https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L180-L183
              // >   See also https://github.com/mikermcneil/machinepack-markdown/blob/5d8cee127e8ce45c702ec9bbb2b4f9bc4b7fafac/machines/parse-docmeta-tags.js#L42-L47
              // >
              // >   e.g. referring to stuff like:
              // >   ```
              // >   <meta name="foo" value="bar">
              // >   <meta name="title" value="Sth with punctuATION and weird CAPS ... but never this long, please">
              // >   ```
              let embeddedMetadata = {};
              for (let tag of (mdString.match(/<meta[^>]*>/igm)||[])) {
                let name = tag.match(/name="([^">]+)"/i)[1];
                let value = tag.match(/value="([^">]+)"/i)[1];
                embeddedMetadata[name] = value;
              }//∞
              if (Object.keys(embeddedMetadata).length >= 1) {
                sails.log.silly(`Parsed ${Object.keys(embeddedMetadata).length} <meta> tags:`, embeddedMetadata);
              }//ﬁ

              // Get last modified timestamp using git, and represent it as a JS timestamp.
              // > Inspired by https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L265-L273
              let lastModifiedAt = (new Date((await sails.helpers.process.executeCommand.with({
                command: `git log -1 --format="%ai" '${path.relative(topLvlRepoPath, pageSourcePath)}'`,
                dir: topLvlRepoPath,
              })).stdout)).getTime();

              // Determine display title (human-readable title) to use for this page.
              let pageTitle;
              if (embeddedMetadata.title) {// Attempt to use custom title, if one was provided.
                if (embeddedMetadata.title.length > 40) {
                  throw new Error(`Failed compiling markdown content: Invalid custom title (<meta name="title" value="${embeddedMetadata.title}">) embedded in "${path.join(topLvlRepoPath, sectionRepoPath)}".  To resolve, try changing the title to a different, valid value, then rebuild.`);
                }//•
                pageTitle = embeddedMetadata.title;
              } else {// Otherwise use the automatically-determined fallback title.
                pageTitle = fallbackPageTitle;
              }


              // If the page has a pageOrderInSection meta tag, we'll use that to sort pages in their bottom level sections.
              let pageOrderInSection;
              // Add handbook pages to pageOrderInSection check, add pageOrderInSection meta tags to each handbook page.
              if(sectionRepoPath === 'docs/') {
                // Set a flag to determine if the page is a readme (e.g. /docs/Using-Fleet/configuration-files/readme.md) or a FAQ page.
                // READMEs in subfolders and FAQ pages don't have pageOrderInSection values, they are always sorted at the end of sections.
                let isPageAReadmeOrFAQ = (_.last(pageUnextensionedLowercasedRelPath.split(/\//)) === 'faq' || _.last(pageUnextensionedLowercasedRelPath.split(/\//)) === 'readme');
                if(embeddedMetadata.pageOrderInSection) {
                  if(isPageAReadmeOrFAQ) {
                  // Throwing an error if a FAQ or README page has a pageOrderInSection meta tag
                    throw new Error(`Failed compiling markdown content: A FAQ or README page has a pageOrderInSection meta tag (<meta name="pageOrderInSection" value="${embeddedMetadata.pageOrderInSection}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, remove this meta tag from the markdown file.`);
                  }
                  // Checking if the meta tag's value is a number higher than 0
                  if (embeddedMetadata.pageOrderInSection <= 0 || _.isNaN(parseInt(embeddedMetadata.pageOrderInSection)) ) {
                    throw new Error(`Failed compiling markdown content: Invalid page rank (<meta name="pageOrderInSection" value="${embeddedMetadata.pageOrderInSection}">) embedded in "${path.join(topLvlRepoPath, sectionRepoPath)}".  To resolve, try changing the rank to a number higher than 0, then rebuild.`);
                  } else {
                    pageOrderInSection = parseInt(embeddedMetadata.pageOrderInSection);
                  }
                } else if(!embeddedMetadata.pageOrderInSection && !isPageAReadmeOrFAQ){
                  // If the page is not a Readme or a FAQ, we'll throw an error if its missing a pageOrderInSection meta tag.
                  throw new Error(`Failed compiling markdown content: A Non FAQ or README Documentation page is missing a pageOrderInSection meta tag (<meta name="pageOrderInSection" value="">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, add a meta tag with a number higher than 0.`);
                }
              }

              // Checking the metadata on /articles pages, and adding the category to the each article's URL.
              if(sectionRepoPath === 'articles/') {
                if(!embeddedMetadata.authorGitHubUsername) {
                  // Throwing an error if the article doesn't have a authorGitHubUsername meta tag
                  throw new Error(`Failed compiling markdown content: An article page is missing a authorGitHubUsername meta tag (<meta name="authorGitHubUsername" value="">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, add a meta tag with the authors GitHub username.`);
                }
                if(!embeddedMetadata.authorFullName) {
                  // Throwing an error if the article doesn't have a authorFullName meta tag
                  throw new Error(`Failed compiling markdown content: An article page is missing a authorFullName meta tag (<meta name="authorFullName" value="">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, add a meta tag with the authors GitHub username.`);
                }
                if(!embeddedMetadata.articleTitle) {
                  // Throwing an error if the article doesn't have a articleTitle meta tag
                  throw new Error(`Failed compiling markdown content: An article page is missing a articleTitle meta tag (<meta name="articleTitle" value="">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, add a meta tag with the title of the article.`);
                }
                if(embeddedMetadata.publishedOn) {
                  if(!embeddedMetadata.publishedOn.match(/^([0-9]{4})-(1[0-2]|0[1-9])-(3[01]|0[1-9]|[12][0-9])$/g)){
                    // Throwing an error if an article page's publishedOn meta value is an invalid ISO date string
                    throw new Error(`Failed compiling markdown content: An article page has an invalid publishedOn value (<meta name="publishedOn" value="${embeddedMetadata.publishedOn}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, add a meta tag with the ISO formatted date the article was published on`);
                  }
                } else {
                  // Throwing an error if the article is missing a 'publishedOn' meta tag
                  throw new Error(`Failed compiling markdown content: An article page is missing a publishedOn meta tag (<meta name="publishedOn" value="2022-04-19">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, add a meta tag with the ISO formatted date the article was published on`);
                }
                if(embeddedMetadata.category) {
                  // Throwing an error if the article has an invalid category.
                  embeddedMetadata.category = embeddedMetadata.category.toLowerCase();
                  let validArticleCategories = ['deploy', 'security', 'engineering', 'success stories', 'announcements', 'guides', 'releases', 'podcasts', 'report' ];
                  if(!validArticleCategories.includes(embeddedMetadata.category)) {
                    throw new Error(`Failed compiling markdown content: An article page has an invalid category meta tag (<meta name="category" value="${embeddedMetadata.category}">) at "${path.join(topLvlRepoPath, pageSourcePath)}". To resolve, change the meta tag to a valid category, one of: ${validArticleCategories}`);
                  }
                } else {
                  // throwing an error if the article is missing a category meta tag
                  throw new Error(`Failed compiling markdown content: An article page is missing a category meta tag (<meta name="category" value="guides">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, add a meta tag with the category of the article`);
                }
                if(embeddedMetadata.articleImageUrl) {
                  // Checking the value of `articleImageUrl` meta tags, and throwing an error if it is not a link to an image.
                  let isValidImage = embeddedMetadata.articleImageUrl.match(/^(https?:\/\/|\.\.)(.+)(\.png|\.jpg|\.jpeg)$/g);
                  if(!isValidImage) {
                    throw new Error(`Failed compiling markdown content: An article page has an invalid a articleImageUrl meta tag (<meta name="articleImageUrl" value="${embeddedMetadata.articleImageUrl}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, change the value of the meta tag to be a URL or repo relative link to an image`);
                  }
                  let isURL = embeddedMetadata.articleImageUrl.match(/https?:\/\/(.+)/g);
                  let isExternal = ! embeddedMetadata.articleImageUrl.match(/https?:\/\/([^\.|blog]+\.)*fleetdm\.com/g);
                  let inWebsiteAssetFolder = embeddedMetadata.articleImageUrl.match(/(?<=\.\.\/website\/assets)(\/images\/(.+))/g);
                  // Modifying the value of the `articleImageUrl` meta tag
                  if (isURL) {
                    if (!isExternal) { // If the image is hosted on fleetdm.com, we'll modify the meta value to reference the file directly in the `website/assets/` folder
                      embeddedMetadata.articleImageUrl = embeddedMetadata.articleImageUrl.replace(/https?:\/\//, '').replace(/^fleetdm\.com/, '');
                    } else { // If the value is a link to an image that is not hosted on fleetdm.com, we won't modify it.
                      // TODO: Enable this error once our Medium articles and their assets have been fully migrated.
                      // throw new Error(`Failed compiling markdown content: An article page has an invalid a articleImageUrl meta tag (<meta name="articleImageUrl" value="${embeddedMetadata.articleImageUrl}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, change the value of the meta tag to be an image that will be hosted on fleetdm.com`);
                    }
                  } else if(inWebsiteAssetFolder) { // If the `articleImageUrl` value is a relative link to the `website/assets/` folder, we'll modify the value to link directly to that folder.
                    embeddedMetadata.articleImageUrl = embeddedMetadata.articleImageUrl.replace(/^\.\.\/website\/assets/g, '');
                  } else { // If the value is not a url and the relative link does not go to the 'website/assets/' folder, we'll throw an error.
                    throw new Error(`Failed compiling markdown content: An article page has an invalid a articleImageUrl meta tag (<meta name="articleImageUrl" value="${embeddedMetadata.articleImageUrl}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, change the value of the meta tag to be a URL or repo relative link to an image in the 'website/assets/images' folder`);
                  }
                }
                // For article pages, we'll attach the category to the `rootRelativeUrlPath`.
                // If the article is categorized as 'product' we'll replace the category with 'use-cases', or if it is categorized as 'success story' we'll replace it with 'device-management'
                rootRelativeUrlPath = (
                  '/' +
                  (embeddedMetadata.category === 'success stories' ? 'device-management' : embeddedMetadata.category === 'security' ? 'securing' : embeddedMetadata.category) + '/' +
                  (pageUnextensionedLowercasedRelPath.split(/\//).map((fileOrFolderName) => encodeURIComponent(fileOrFolderName.replace(/^[0-9]+[\-]+/,''))).join('/'))
                );
              }

              // Determine unique HTML id
              // > • This will become the filename of the resulting HTML.
              let htmlId = (
                sectionRepoPath.slice(0,10)+
                '--'+
                _.last(pageUnextensionedLowercasedRelPath.split(/\//)).slice(0,20)+
                '--'+
                sails.helpers.strings.random.with({len:10})// if two files in different folders happen to have the same filename, there is a 1/16^10 chance of a collision (this is small enough- worst case, the build fails at the uniqueness check and we rerun it.)
              ).replace(/[^a-z0-9\-]/ig,'');

              // Generate HTML file
              let htmlOutputPath = path.resolve(sails.config.appPath, path.join(APP_PATH_TO_COMPILED_PAGE_PARTIALS, htmlId+'.ejs'));
              if (dry) {
                sails.log('Dry run: Would have generated file:', htmlOutputPath);
              } else {
                await sails.helpers.fs.write(htmlOutputPath, htmlString);
              }

              // Determine the path of the file in the fleet repo so we can link to
              // the file on github from fleetdm.com (e.g. Using-Fleet/fleetctl-CLI.md)
              let sectionRelativeRepoPath = path.relative(path.join(topLvlRepoPath, sectionRepoPath), path.resolve(pageSourcePath));

              // Append to what will become configuration for the Sails app.
              builtStaticContent.markdownPages.push({
                url: rootRelativeUrlPath,
                title: pageTitle,
                lastModifiedAt: lastModifiedAt,
                htmlId: htmlId,
                pageOrderInSectionPath: pageOrderInSection,
                sectionRelativeRepoPath: sectionRelativeRepoPath,
                meta: _.omit(embeddedMetadata, ['title', 'pageOrderInSection']),
                linksForHandbookIndex: linksForHandbookIndex.length > 0 ? linksForHandbookIndex : undefined,
              });
            }
          }//∞ </each source file>
        }//∞ </each section repo path>

        // Attach partials dir path in what will become configuration for the Sails app.
        // (This is for easier access later, without defining this constant in more than one place.)
        builtStaticContent.compiledPagePartialsAppPath = APP_PATH_TO_COMPILED_PAGE_PARTIALS;

      },
    ]);

    //  ██████╗ ███████╗██████╗ ██╗      █████╗  ██████╗███████╗       ███████╗ █████╗ ██╗██╗     ███████╗██████╗  ██████╗
    //  ██╔══██╗██╔════╝██╔══██╗██║     ██╔══██╗██╔════╝██╔════╝       ██╔════╝██╔══██╗██║██║     ██╔════╝██╔══██╗██╔════╝██╗
    //  ██████╔╝█████╗  ██████╔╝██║     ███████║██║     █████╗         ███████╗███████║██║██║     ███████╗██████╔╝██║     ╚═╝
    //  ██╔══██╗██╔══╝  ██╔═══╝ ██║     ██╔══██║██║     ██╔══╝         ╚════██║██╔══██║██║██║     ╚════██║██╔══██╗██║     ██╗
    //  ██║  ██║███████╗██║     ███████╗██║  ██║╚██████╗███████╗    ██╗███████║██║  ██║██║███████╗███████║██║  ██║╚██████╗╚═╝
    //  ╚═╝  ╚═╝╚══════╝╚═╝     ╚══════╝╚═╝  ╚═╝ ╚═════╝╚══════╝    ╚═╝╚══════╝╚═╝  ╚═╝╚═╝╚══════╝╚══════╝╚═╝  ╚═╝ ╚═════╝
    //
    // Replace .sailsrc file.
    // > This takes the compiled menu file from doc-templater and injects it into the .sailsrc file so it
    // > can be accessed for the purposes of config using `sails.config.builtStaticContent`.
    if (dry) {
      sails.log('Dry run: Would have folded the following onto .sailsrc as "builtStaticContent":', builtStaticContent);
    } else {
      let sailsrcPath = path.resolve(sails.config.appPath, '.sailsrc');
      let oldSailsrcJson = await sails.helpers.fs.readJson(sailsrcPath);
      await sails.helpers.fs.writeJson.with({
        force: true,
        destination: sailsrcPath,
        json: {
          ...oldSailsrcJson,
          builtStaticContent: builtStaticContent,
        }
      });
    }

  }


};

module.exports = {


  friendlyName: 'Build static content',


  description: 'Generate HTML partials from source files in fleetdm/fleet repo (e.g. docs in markdown, or queries in YAML), and configure metadata about the generated files so it is available in `sails.config.builtStaticContent`.',


  inputs: {
    dry: { type: 'boolean', description: 'Whether to make this a dry run.  (.sailsrc file will not be overwritten.  HTML files will not be generated.)' },
    skipGithubRequests: { type: 'boolean', description: 'Whether to minimize requests to the GitHub API which usually can be skipped during local development, such as requests used for fetching GitHub avatar URLs'},
    githubAccessToken: { type: 'string', description: 'If provided, A GitHub token will be used to authenticate requests to the GitHub API'},
  },


  fn: async function ({ dry, skipGithubRequests, githubAccessToken }) {

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
          query.requiresMdm = false;
          if (query.tags) {
            if(!_.isString(query.tags)) {
              queriesWithProblematicTags.push(query);
            } else {
              // Splitting tags into an array to format them.
              let tagsToFormat = query.tags.split(',');
              let formattedTags = [];
              for (let tag of tagsToFormat) {
                if(tag !== '') {// « Ignoring any blank tags caused by trailing commas in the YAML.
                  // If a query has a 'requires MDM' tag, we'll set requiresMDM to true for this query, and we'll ingore this tag.
                  if(_.trim(tag.toLowerCase()) === 'mdm required'){
                    query.requiresMdm = true;
                  } else {
                    // Removing any extra whitespace from tags and changing them to be in lower case.
                    formattedTags.push(_.trim(tag.toLowerCase()));
                  }
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

        let githubDataByUsername = {};

        if(skipGithubRequests) {// If the --skipGithubRequests flag was provided, we'll skip querying GitHubs API
          sails.log('Skipping GitHub API requests for contributer profiles.\nNOTE: The contributors in the standard query library will be populated with fake data. To see how the standard query library will look on fleetdm.com, run this script without the `--skipGithubRequests` flag.');
          // Because we're not querying GitHub to get the real names for contributer profiles, we'll use their GitHub username as their name and their handle
          for (let query of queries) {
            let usernames = query.contributors.split(',');
            let contributorProfiles = [];
            for (let username of usernames) {
              contributorProfiles.push({
                name: username,
                handle: username,
                avatarUrl: 'https://placekitten.com/200/200',
                htmlUrl: 'https://github.com/'+encodeURIComponent(username),
              });
            }
            query.contributors = contributorProfiles;
          }
        } else {// If the --skipGithubRequests flag was not provided, we'll query GitHub's API to get additional information about each contributor.

          let baseHeadersForGithubRequests = {
            'User-Agent': 'Fleet-Standard-Query-Library',
            'Accept': 'application/vnd.github.v3+json',
          };

          if(githubAccessToken) {
            // If a GitHub access token was provided, add it to the baseHeadersForGithubRequests object.
            baseHeadersForGithubRequests['Authorization'] = `token ${githubAccessToken}`;
          }
          await sails.helpers.flow.simultaneouslyForEach(githubUsernames, async(username)=>{
            githubDataByUsername[username] = await sails.helpers.http.get.with({
              url: 'https://api.github.com/users/' + encodeURIComponent(username),
              headers: baseHeadersForGithubRequests,
            }).catch((err)=>{// If the above GET requests return a non 200 response we'll look for signs that the user has hit their GitHub API rate limit.
              if (err.raw.statusCode === 403 && err.raw.headers['x-ratelimit-remaining'] === '0') {// If the user has reached their GitHub API rate limit, we'll throw an error that suggest they run this script with the `--skipGithubRequests` flag.
                throw new Error('GitHub API rate limit exceeded. If you\'re running this script in a development environment, use the `--skipGithubRequests` flag to skip querying the GitHub API. See full error for more details:\n'+err);
              } else {// If the error was not because of the user's API rate limit, we'll display the full error
                throw err;
              }
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
            let pageUnextensionedUnwhitespacedLowercasedRelPath = (
              pageRelSourcePath
              .replace(/(^|\/)([^/]+)\.[^/]*$/, '$1$2')
              .split(/\//).map((fileOrFolderName) => fileOrFolderName.toLowerCase().replace(/\s+/g, '-')).join('/')
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

            // Determine URL for this page.
            // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
            // Note: If this is an article page, then the rootRelativeUrlPath will be further modified below to
            // also include the article's category: https://github.com/fleetdm/fleet/blob/9d2acb5751f4cebfb211ae1f15c9289143b5b79c/website/scripts/build-static-content.js#L441-L447
            // > TODO: Try eliminating this exception by moving the processing of all page metadata upwards, so
            // > that it occurs before this spot in the file, so that all URL-determining logic can happen here,
            // > all in one place.  (For more context, see https://github.com/fleetdm/confidential/issues/1537 )
            // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
            let rootRelativeUrlPath = (
              (
                SECTION_INFOS_BY_SECTION_REPO_PATHS[sectionRepoPath].urlPrefix +
                '/' + (
                  pageUnextensionedUnwhitespacedLowercasedRelPath
                  .split(/\//).map((fileOrFolderName) => encodeURIComponent(fileOrFolderName.replace(/^[0-9]+[\-]+/,''))).join('/')// « Get URL-friendly by encoding characters and stripping off ordering prefixes (like the "1-" in "1-Using-Fleet") for all folder and file names in the path.
                )
              ).replace(RX_README_FILENAME, '')// « Interpret README files as special and map it to the URL representing its containing folder.
            );

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
              // > • What about images referenced in markdown files? :: For documentation and handbook files, they need to be referenced using an absolute URL of the src-- e.g. ![](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/foo.png). For articles, you can use the absolute URL of the src - e.g. ![](https://fleetdm.com/images/articles/foo.png) OR the relative repo path e.g. ![](../website/assets/images/articles/foo.png). See also https://github.com/fleetdm/fleet/issues/706#issuecomment-884641081 for reasoning.
              // > • What about GitHub-style emojis like `:white_check_mark:`?  :: Use actual unicode emojis instead.  Need to revisit this?  Visit https://github.com/fleetdm/fleet/pull/1380/commits/19a6e5ffc70bf41569293db44100e976f3e2bda7 for more info.
              let mdString = await sails.helpers.fs.read(pageSourcePath);

              // Look for non example @fleetdm.com email addresses in the Markdown string, if any are found, throw an error.
              if(mdString.match(/[A-Z0-9._%+-]+@fleetdm\.com/gi)) {
                throw new Error(`A Markdown file (${pageSourcePath}) contains a @fleetdm.com email address. To resolve this error, remove the email address in that file or change it to be an @example.com email address and try running this script again.`);
              }
              // Look for anything in markdown content that could be interpreted as a Vue template when converted to HTML (e.g. {{ foo }}). If any are found, throw an error.
              if(mdString.match(/\{\{([^}]+)\}\}/gi)) {
                throw new Error(`A Markdown file (${pageSourcePath}) contains a Vue template (${mdString.match(/\{\{([^}]+)\}\}/gi)[0]}) that will cause client-side javascript errors when converted to HTML. To resolve this error, change or remove the double curly brackets in this file.`);
              }
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
                  (path.relative(topLvlRepoPath, referencedPageSourcePath).replace(/(^|\/)([^/]+)\.[^/]*$/, '$1$2').split(/\//).map((fileOrFolderName) => fileOrFolderName.toLowerCase().replace(/\s+/g, '-')).join('/'))
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
                    return hrefString.replace(/href="https?:\/\//, '').replace(/^([^\.]+\.)*fleetdm\.com/, 'href="');
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
              try {
                for (let tag of (mdString.match(/<meta[^>]*>/igm)||[])) {
                  let name = tag.match(/name="([^">]+)"/i)[1];
                  let value = tag.match(/value="([^">]+)"/i)[1];
                  embeddedMetadata[name] = value;
                }//∞
              } catch (err) {
                throw new Error(`An error occured while parsing <meta> tags in Markdown in "${path.join(topLvlRepoPath, pageSourcePath)}". Tip: Check the markdown being changed and make sure it doesn\'t contain any code snippets with <meta> inside, as this can fool the build script. Full error: ${err.message}`);
              }
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
                  throw new Error(`Failed compiling markdown content: Invalid custom title (<meta name="title" value="${embeddedMetadata.title}">) embedded in "${path.join(topLvlRepoPath, sectionRepoPath)}".  To resolve, try changing the title to a shorter (less than 40 characters), valid value, then rebuild.`);
                }//•
                pageTitle = embeddedMetadata.title;
              } else {// Otherwise use the automatically-determined fallback title.
                pageTitle = fallbackPageTitle;
              }


              // If the page has a pageOrderInSection meta tag, we'll use that to sort pages in their bottom level sections.
              let pageOrderInSection;
              let docNavCategory;
              if(sectionRepoPath === 'docs/') {
                // Set a flag to determine if the page is a readme (e.g. /docs/Using-Fleet/configuration-files/readme.md) or a FAQ page.
                // READMEs in subfolders and FAQ pages don't have pageOrderInSection values, they are always sorted at the end of sections.
                let isPageAReadmeOrFAQ = (_.last(pageUnextensionedUnwhitespacedLowercasedRelPath.split(/\//)) === 'faq' || _.last(pageUnextensionedUnwhitespacedLowercasedRelPath.split(/\//)) === 'readme');
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
                if(embeddedMetadata.navSection){
                  docNavCategory = embeddedMetadata.navSection;
                } else {
                  docNavCategory = 'Uncategorized';
                }
              }

              if(sectionRepoPath === 'handbook/') {
                if(!embeddedMetadata.maintainedBy) {
                  // Throw an error if a handbook page is missing a maintainedBy meta tag.
                  throw new Error(`Failed compiling markdown content: A handbook page is missing a maintainedBy meta tag (<meta name="maintainedBy" value="">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, add a maintainedBy meta tag with the page maintainer's GitHub username as the value.`);
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
                if(!embeddedMetadata.category) {
                  // Throwing an error if the article is missing a category meta tag
                  throw new Error(`Failed compiling markdown content: An article page is missing a category meta tag (<meta name="category" value="guides">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, add a meta tag with the category of the article`);
                } else {
                  // Throwing an error if the article has an invalid category.
                  let validArticleCategories = ['deploy', 'security', 'engineering', 'success stories', 'announcements', 'guides', 'releases', 'podcasts', 'report' ];
                  if(!validArticleCategories.includes(embeddedMetadata.category)) {
                    throw new Error(`Failed compiling markdown content: An article page has an invalid category meta tag (<meta name="category" value="${embeddedMetadata.category}">) at "${path.join(topLvlRepoPath, pageSourcePath)}". To resolve, change the meta tag to a valid category, one of: ${validArticleCategories}`);
                  }
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
                    } else { // If the value is a link to an image that will not be hosted on fleetdm.com, we'll throw an error.
                      throw new Error(`Failed compiling markdown content: An article page has an invalid a articleImageUrl meta tag (<meta name="articleImageUrl" value="${embeddedMetadata.articleImageUrl}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, change the value of the meta tag to be an image that will be hosted on fleetdm.com`);
                    }
                  } else if(inWebsiteAssetFolder) { // If the `articleImageUrl` value is a relative link to the `website/assets/` folder, we'll modify the value to link directly to that folder.
                    embeddedMetadata.articleImageUrl = embeddedMetadata.articleImageUrl.replace(/^\.\.\/website\/assets/g, '');
                  } else { // If the value is not a url and the relative link does not go to the 'website/assets/' folder, we'll throw an error.
                    throw new Error(`Failed compiling markdown content: An article page has an invalid a articleImageUrl meta tag (<meta name="articleImageUrl" value="${embeddedMetadata.articleImageUrl}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, change the value of the meta tag to be a URL or repo relative link to an image in the 'website/assets/images' folder`);
                  }
                }
                if(embeddedMetadata.description && embeddedMetadata.description.length > 150) {
                  // Throwing an error if the article's description meta tag value is over 150 characters long
                  throw new Error(`Failed compiling markdown content: An article page has an invalid description meta tag (<meta name="description" value="${embeddedMetadata.description}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, make sure the value of the meta description is less than 150 characters long.`);
                }
                // For article pages, we'll attach the category to the `rootRelativeUrlPath`.
                // If the article is categorized as 'product' we'll replace the category with 'use-cases', or if it is categorized as 'success story' we'll replace it with 'device-management'
                rootRelativeUrlPath = (
                  '/' +
                  (encodeURIComponent(embeddedMetadata.category === 'success stories' ? 'success-stories' : embeddedMetadata.category === 'security' ? 'securing' : embeddedMetadata.category)) + '/' +
                  (pageUnextensionedUnwhitespacedLowercasedRelPath.split(/\//).map((fileOrFolderName) => encodeURIComponent(fileOrFolderName.replace(/^[0-9]+[\-]+/,''))).join('/'))
                );
              }

              // Assert uniqueness of URL paths.
              if (rootRelativeUrlPathsSeen.includes(rootRelativeUrlPath)) {
                throw new Error('Failed compiling markdown content: Files as currently named would result in colliding (duplicate) URLs for the website.  To resolve, rename the pages whose names are too similar.  Duplicate detected: ' + rootRelativeUrlPath);
              }//•
              rootRelativeUrlPathsSeen.push(rootRelativeUrlPath);

              // Determine unique HTML id
              // > • This will become the filename of the resulting HTML.
              let htmlId = (
                sectionRepoPath.slice(0,10)+
                '--'+
                _.last(pageUnextensionedUnwhitespacedLowercasedRelPath.split(/\//)).slice(0,20)+
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
                docNavCategory: docNavCategory ? docNavCategory : undefined,// FUTURE: No docs specific markdown page attributes.
                sectionRelativeRepoPath: sectionRelativeRepoPath,
                meta: _.omit(embeddedMetadata, ['title', 'pageOrderInSection']),
                linksForHandbookIndex: linksForHandbookIndex.length > 0 ? linksForHandbookIndex : undefined,
              });
            }
          }//∞ </each source file>
        }//∞ </each section repo path>

        // After we build the Markdown pages, we'll merge the osquery schema with the Fleet schema overrides, then create EJS partials for each table in the merged schema.

        let expandedTables = await sails.helpers.getExtendedOsquerySchema();

        // Once we have our merged schema, we'll create ejs partials for each table.
        for(let table of expandedTables) {
          let keywordsForSyntaxHighlighting = [];
          keywordsForSyntaxHighlighting.push(table.name);
          if(!table.hidden) { // If a table has `"hidden": true` the table won't be shown in the final schema, and we'll ignore it
            // Start building the markdown string for this table.
            let tableMdString = '\n## '+table.name;
            if(table.evented){
              // If the table has `"evented": true`, we'll add an evented table label (in html)
              tableMdString += '   <a style="text-decoration: none" purpose="evented-table-label" href="https://fleetdm.com/guides/osquery-evented-tables-overview?utm_source=fleetdm.com&utm_content=table-'+_.escape(encodeURIComponent(table.name))+'"><span>EVENTED TABLE</span></a>\n';
            }
            // Add the tables description to the markdown string and start building the table in the markdown string
            tableMdString += '\n\n'+table.description+'\n\n|Column | Type | Description |\n|-|-|-|\n';

            // Iterate through the columns of the table, we'll add a row to the markdown table element for each column in this schema table
            for(let column of table.columns) {
              if(!column.hidden) { // If te column is hidden, we won't add it to the final table.
                let columnDescriptionForTable = '';// Set the initial value of the description that will be added to the table for this column.
                if(column.description) {
                  columnDescriptionForTable = column.description;
                }
                // Replacing pipe characters and newlines with html entities in column descriptions to keep it from breaking markdown tables.
                columnDescriptionForTable = columnDescriptionForTable.replace(/\|/g, '&#124;').replace(/\n/gm, '&#10;');

                keywordsForSyntaxHighlighting.push(column.name);
                if(column.required) { // If a column has `"required": true`, we'll add a note to the description that will be added to the table
                  columnDescriptionForTable += '<br> **Required in `WHERE` clause** ';
                }
                if(column.requires_user_context) { // If a column has `"requires_user_context": true`, we'll add a note to the description that will be added to the table
                  columnDescriptionForTable += '<br> **Defaults to root** &nbsp;&nbsp;[Learn more](https://fleetdm.com/guides/osquery-consider-joining-against-the-users-table?utm_source=fleetdm.com&utm_content=table-'+encodeURIComponent(table.name)+')';
                }
                if(column.platforms) { // If a column has an array of platforms, we'll add a note to the final column description

                  let platformString = '<br> **Only available on ';// start building a string to add to the column's description

                  if(column.platforms.length > 3) {// FUTURE: add support for more than three platform values in columns.
                    throw new Error('Support for more than three platforms has not been implemented yet.');
                  }

                  if(column.platforms.length === 3) { // Because there are only four options for platform, we can safely assume that there will be at most 3 platforms, so we'll just handle this one of three ways
                    // If there are three, we'll add a string with an oxford comma. e.g., "On macOS, Windows, and Linux"
                    platformString += `${column.platforms[0]}, ${column.platforms[1]}, and ${column.platforms[2]}`;
                  } else if(column.platforms.length === 2) {
                    // If there are two values in the platforms array, it will be formated as "[Platform 1] and [Platform 2]"
                    platformString += `${column.platforms[0]} and ${column.platforms[1]}`;
                  } else {
                    // Otherwise, there is only one value in the platform array and we'll add that value to the column's description
                    platformString += column.platforms[0];
                  }
                  platformString += '** ';
                  columnDescriptionForTable += platformString; // Add the platform string to the column's description.
                }
                tableMdString += ' | '+column.name+' | '+ column.type +' | '+columnDescriptionForTable+'|\n';
              }
            }
            if(table.examples) { // If this table has a examples value (These will be in the Fleet schema JSON) We'll add the examples to the markdown string.
              tableMdString += '\n### Example\n\n'+table.examples+'\n';
            }
            if(table.notes) { // If this table has a notes value (These will be in the Fleet schema JSON) We'll add the notes to the markdown string.
              tableMdString += '\n### Notes\n\n'+table.notes+'\n';
            }
            // Determine the htmlId for table
            let htmlId = (
              'table--'+
              table.name+
              '--'+
              sails.helpers.strings.random.with({len:10})
            ).replace(/[^a-z0-9\-]/ig,'');

            // Convert the markdown string to HTML.
            let htmlString = await sails.helpers.strings.toHtml.with({mdString: tableMdString, addIdsToHeadings: false});

            // Add the language-sql class to codeblocks in generated HTML partial for syntax highlighting.
            htmlString = htmlString.replace(/(<pre><code)([^>]*)(>)/gm, '$1 class="language-sql"$2$3');

            htmlString = htmlString.replace(/(href="https?:\/\/([^"]+)")/g, (hrefString)=>{// « Modify links that are potentially external
              // Check if this is an external link (like https://google.com) but that is ALSO not a link
              // to some page on the destination site where this will be hosted, like `(*.)?fleetdm.com`.
              // If external, add target="_blank" so the link will open in a new tab.
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

            // Determine the path of the folder where the built HTML partials will live.
            let htmlOutputPath = path.resolve(sails.config.appPath, path.join(APP_PATH_TO_COMPILED_PAGE_PARTIALS, htmlId+'.ejs'));
            if (dry) {
              sails.log('Dry run: Would have generated file:', htmlOutputPath);
            } else {
              await sails.helpers.fs.write(htmlOutputPath, htmlString);
            }
            // Add this table to the array of schemaTables in builtStaticContent.
            builtStaticContent.markdownPages.push({
              url: '/tables/'+encodeURIComponent(table.name),
              title: table.name,
              htmlId: htmlId,
              evented: table.evented,
              platforms: table.platforms,
              keywordsForSyntaxHighlighting: keywordsForSyntaxHighlighting,
              sectionRelativeRepoPath: table.name, // Setting the sectionRelativeRepoPath to an arbitrary string to work with existing pages.
              githubUrl: table.fleetRepoUrl,
            });
          }
        }
        // Attach partials dir path in what will become configuration for the Sails app.
        // (This is for easier access later, without defining this constant in more than one place.)
        builtStaticContent.compiledPagePartialsAppPath = APP_PATH_TO_COMPILED_PAGE_PARTIALS;

      },
      async()=>{
        // Validate the pricing table yaml and add it to builtStaticContent.pricingTable.
        let RELATIVE_PATH_TO_PRICING_TABLE_YML_IN_FLEET_REPO = 'handbook/product/pricing-features-table.yml';// TODO: Is there a better home for this file?
        let yaml = await sails.helpers.fs.read(path.join(topLvlRepoPath, RELATIVE_PATH_TO_PRICING_TABLE_YML_IN_FLEET_REPO)).intercept('doesNotExist', (err)=>new Error(`Could not find pricing table features YAML file at "${RELATIVE_PATH_TO_PRICING_TABLE_YML_IN_FLEET_REPO}".  Was it accidentally moved?  Raw error: `+err.message));
        let pricingTableCategories = YAML.parse(yaml, {prettyErrors: true});

        for(let category of pricingTableCategories){
          if(!category.categoryName){ // Throw an error if a category is missing a categoryName.
            throw new Error('Could not build pricing table config from pricing-features-table.yml, a category in the pricing table configuration is missing a categoryName. To resolve, make sure every category in the pricing table YAML file has a categoryName');
          }
          if(!category.features){// Throw an error if a category is missing `features`.
            throw new Error('Could not build pricing table config from pricing-features-table.yml, the "'+category.categoryName+'" category in the yaml file is missing features. To resolve, add an array of features to this category.');
          }
          if(!_.isArray(category.features)){ // Throw an error if a category's `features`` is not an array.
            throw new Error('Could not build pricing table config from pricing-features-table.yml, The value of the "'+category.categoryName+'" category is invalid, to resolve, change the features for this category to be an array of objects.');
          }
          // Validate all features in a category.
          for(let feature of category.features){
            if(!feature.name) { // Throw an error if a feature is missing a `name`.
              throw new Error('Could not build pricing table config from pricing-features-table.yml. A feature in the "'+category.categoryName+'" category is missing a "name". To resolve, add a "name" to this feature '+feature);
            }
            if(!feature.tier) { // Throw an error if a feature is missing a `tier`.
              throw new Error('Could not build pricing table config from pricing-features-table.yml. The "'+feature.name+'" feature is missing a "tier". To resolve, add a "tier" (either "Free" or "Premium") to this feature.');
            } else if(!_.contains(['Free', 'Premium'], feature.tier)){ // Throw an error if a feature's `tier` is not "Free" or "Premium".
              throw new Error('Could not build pricing table config from pricing-features-table.yml. The "'+feature.name+'" feature has an invalid "tier". to resolve, change the value of this features "tier" (currently set to '+feature.tier+') to be either "Free" or "Premium".');
            }
            if(feature.comingSoon === undefined) { // Throw an error if a feature is missing a `comingSoon` value
              throw new Error('Could not build pricing table config from pricing-features-table.yml. The "'+feature.name+'" feature is missing a "comingSoon" value (boolean). To resolve, add a comingSoon value to this feature.');
            } else if(typeof feature.comingSoon !== 'boolean'){ // Throw an error if the `comingSoon` value is not a boolean.
              throw new Error('Could not build pricing table config from pricing-features-table.yml. The "'+feature.name+'" feature has an invalid "comingSoon" value (currently set to '+feature.comingSoon+'). To resolve, change the value of "comingSoon" for this feature to be either "true" or "false".');
            }
          }
        }
        builtStaticContent.pricingTable = pricingTableCategories;
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

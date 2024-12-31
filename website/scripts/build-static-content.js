module.exports = {


  friendlyName: 'Build static content',


  description: 'Generate HTML partials from source files in fleetdm/fleet repo (e.g. docs in markdown, or queries in YAML), and configure metadata about the generated files so it is available in `sails.config.builtStaticContent`.',


  inputs: {
    dry: { type: 'boolean', description: 'Whether to make this a dry run.  (.sailsrc file will not be overwritten.  HTML files will not be generated.)' },
    githubAccessToken: { type: 'string', description: 'If provided, A GitHub token will be used to authenticate requests to the GitHub API'},
  },


  fn: async function ({ dry, githubAccessToken }) {
    let path = require('path');
    let YAML = require('yaml');
    let util = require('util');
    // FUTURE: If we ever need to gather source files from other places or branches, etc, see git history of this file circa 2021-05-19 for an example of a different strategy we might use to do that.
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    // The data we're compiling will get built into this dictionary and then written on top of the .sailsrc file.
    let builtStaticContent = {};
    let rootRelativeUrlPathsSeen = [];
    let baseHeadersForGithubRequests;

    if(githubAccessToken) {// If a github token was provided, set headers for requests to GitHub.
      baseHeadersForGithubRequests = {
        'User-Agent': 'Fleet-Standard-Query-Library',
        'Accept': 'application/vnd.github.v3+json',
        'Authorization': `token ${githubAccessToken}`,
      };
    } else {
      sails.log('Skipping GitHub API requests for contributer profiles and ritual validation.\nNOTE: The contributors in the standard query library will be populated with fake data.\nTo see how the standard query library will look on fleetdm.com, pass a GitHub access token into this script with the `--githubAccessToken={YOUR_GITHUB_ACCESS_TOKEN}` flag. \n Note: This script can take up to 30s to run.');
    }//ﬁ

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
          // Remove the platform name from query names. This allows us to keep queries at their existing URLs while hiding them in the UI.
          query.name = query.name.replace(/\s\(macOS\)|\(Windows\)|\(Linux\)$/, '');
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
                  } else if(_.trim(tag.toLowerCase()) === 'critical') {
                    query.critical = true;
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

        // If a GitHub access token was provided, validate all users listed in the standard query library YAML.
        if(githubAccessToken) {
          await sails.helpers.flow.simultaneouslyForEach(githubUsernames, async(username)=>{
            githubDataByUsername[username] = await sails.helpers.http.get.with({
              url: 'https://api.github.com/users/' + encodeURIComponent(username),
              headers: baseHeadersForGithubRequests,
            }).intercept((err)=>{
              return new Error(`When validating users in standard-query-library.yml, an error when a request was sent to GitHub get the information about a user (username: ${username}). Error: ${err.stack}`);
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
        } else {// Otherwise, use the Github username as contributor's names and handles and use fake profile pictures.
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
        }//ﬁ

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

            // If this page is in the docs/contributing/ folder, skip it.
            if(sectionRepoPath === 'docs/' && _.startsWith(pageUnextensionedUnwhitespacedLowercasedRelPath, 'contributing/')){
              continue;
            }
            // Skip pages in folders starting with an underscore character.
            if(sectionRepoPath === 'docs/' &&  _.startsWith(pageRelSourcePath.split(/\//).slice(-2)[0], '_')){
              continue;
            }
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
              // Look for multi-line HTML comments starting after lists without a blank newline. (The opening comment block is parsed as part of the list item preceeding it, and the closing block will be parsed as a paragraph)
              if(mdString.match(/(-|\d\.)\s.*\n<!-+\n/g)) {
                throw new Error(`A Markdown file (${pageSourcePath}) contains an HTML comment directly after a list that will cause rendering issues when converted to HTML. To resolve this error, add a blank newline before the start of the HTML comment in this file.`);
              }
              // Look for anything in markdown content that could be interpreted as a Vue template when converted to HTML (e.g. {{ foo }}). If any are found, throw an error.
              if(mdString.match(/\{\{([^}]+)\}\}/gi)) {
                throw new Error(`A Markdown file (${pageSourcePath}) contains a Vue template (${mdString.match(/\{\{([^}]+)\}\}/gi)[0]}) that will cause client-side javascript errors when converted to HTML. To resolve this error, change or remove the double curly brackets in this file.`);
              }
              mdString = mdString.replace(/(<call-to-action[\s\S]+[^>\n+])\n+(>)/g, '$1$2'); // « Removes any newlines that might exist before the closing `>` when the <call-to-action> compontent is added to markdown files.
              // [?] Looking for code that used to be here related to syntax highlighting?  Please see https://github.com/fleetdm/fleet/pull/14124/files  -mikermcneil, 2023-09-25
              let htmlString = await sails.helpers.strings.toHtml(mdString);
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
                // Throw an error if any relative links containing hash links are missing the Markdown page's extension.
                let pageContainsInvalidRelativeHashLink = oldRelPath.match(/[^(\.md)]#([^/]*$)/);
                if(pageContainsInvalidRelativeHashLink){
                  throw new Error(`Could not build HTML partials from Markdown. A page (${pageRelSourcePath}) contains an invalid relative Markdown link (${hrefString.replace(/^href=/, '')}). To resolve, make sure all relative links on this page that link to a specific section include the Markdown page extension (.md) and try running this script again.`);
                }
                let referencedPageNewUrl = 'https://fleetdm.com/' + (
                  (path.relative(topLvlRepoPath, referencedPageSourcePath).replace(/(^|\/)([^/]+)\.[^/]*$/, '$1$2').split(/\//)
                  .map((fileOrFolderNameWithPossibleUrlEncodedSpaces) => fileOrFolderNameWithPossibleUrlEncodedSpaces.toLowerCase().replace(/%20/g, '-'))// « Replaces url-encoded spaces with dashes to support relative links to folders with spaces on Github.com and the Fleet website.
                  .map((fileOrFolderNameWithPossibleSpaces) => fileOrFolderNameWithPossibleSpaces.toLowerCase().replace(/\s/g, '-')).join('/'))// « Replaces spaces in folder names with dashes to support relative links to folders with spaces on Fleet website.
                  .split(/\//).map((fileOrFolderNameWithNoSpaces) => encodeURIComponent(fileOrFolderNameWithNoSpaces.replace(/^[0-9]+[\-]+/,''))).join('/')
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
                // Note: links to trust.fleetdm.com and blog.fleetdm.com will be treated as an external link.
                let isExternal = ! hrefString.match(/^href=\"https?:\/\/([^\.|trust|blog]+\.)*fleetdm\.com/g);// « FUTURE: make this smarter with sails.config.baseUrl + _.escapeRegExp()
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
              // > Note: These meta tags are parsed from the HTML generated from markdown to prevent reading <meta> tags in code examples.
              // > This works because HTML in Markdown files is added as-is, while any <meta> tags in codeblocks would have their brackets replaced with HTML entities when they are converted to HTML.
              let embeddedMetadata = {};
              try {
                for (let tag of (htmlString.match(/<meta[^>]*>/igm)||[])) {
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
              let lastModifiedAt;
              if(!githubAccessToken) {
                lastModifiedAt = Date.now();
              } else if(process.env.GITHUB_REF_NAME && process.env.GITHUB_REF_NAME === 'main') {// Only add lastModifiedAt timestamps if this test is running on the main branch
                let responseData = await sails.helpers.http.get.with({// [?]: https://docs.github.com/en/rest/commits/commits?apiVersion=2022-11-28#list-commits
                  url: 'https://api.github.com/repos/fleetdm/fleet/commits',
                  data: {
                    path: path.join(sectionRepoPath, pageRelSourcePath),
                    page: 1,
                    per_page: 1,//eslint-disable-line camelcase
                  },
                  headers: baseHeadersForGithubRequests,
                }).intercept((err)=>{
                  return new Error(`When getting the commit history for ${path.join(sectionRepoPath, pageRelSourcePath)} to get a lastModifiedAt timestamp, an error occured.`, err);
                });
                // The value we'll use for the lastModifiedAt timestamp will be date value of the `commiter` property of the `commit` we got in the API response from github.
                let mostRecentCommitToThisFile = responseData[0];
                if(!mostRecentCommitToThisFile.commit || !mostRecentCommitToThisFile.commit.committer) {
                  // Throw an error if the the response from GitHub is missing a commit or commiter.
                  throw new Error(`When getting the commit history for ${path.join(sectionRepoPath, pageRelSourcePath)} to get a lastModifiedAt timestamp, the response from the GitHub API did not include information about the most recent commit. Response from GitHub: ${util.inspect(responseData, {depth:null})}`);
                }
                lastModifiedAt = (new Date(mostRecentCommitToThisFile.commit.committer.date)).getTime(); // Convert the UTC timestamp from GitHub to a JS timestamp.
              }

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
                      throw new Error(`Failed compiling markdown content: An article page has an invalid articleImageUrl meta tag (<meta name="articleImageUrl" value="${embeddedMetadata.articleImageUrl}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, change the value of the meta tag to be an image that will be hosted on fleetdm.com`);
                    }
                  } else if(inWebsiteAssetFolder) { // If the `articleImageUrl` value is a relative link to the `website/assets/` folder, we'll modify the value to link directly to that folder.
                    embeddedMetadata.articleImageUrl = embeddedMetadata.articleImageUrl.replace(/^\.\.\/website\/assets/g, '');
                  } else { // If the value is not a url and the relative link does not go to the 'website/assets/' folder, we'll throw an error.
                    throw new Error(`Failed compiling markdown content: An article page has an invalid articleImageUrl meta tag (<meta name="articleImageUrl" value="${embeddedMetadata.articleImageUrl}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, change the value of the meta tag to be a URL or repo relative link to an image in the 'website/assets/images' folder`);
                  }
                }
                if(embeddedMetadata.description && embeddedMetadata.description.length > 150) {
                  // Throwing an error if the article's description meta tag value is over 150 characters long
                  throw new Error(`Failed compiling markdown content: An article page has an invalid description meta tag (<meta name="description" value="${embeddedMetadata.description}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, make sure the value of the meta description is less than 150 characters long.`);
                }
                if(embeddedMetadata.showOnTestimonialsPageWithEmoji){
                  // Throw an error if a showOnTestimonialsPageWithEmoji value is not one of: 🥀, 🔌, 🚪, or 🪟.
                  if(!['🥀', '🔌', '🚪', '🪟'].includes(embeddedMetadata.showOnTestimonialsPageWithEmoji)){
                    throw new Error(`Failed compiling markdown content: An article page has an invalid showOnTestimonialsPageWithEmoji meta tag (<meta name="showOnTestimonialsPageWithEmoji" value="${embeddedMetadata.articleImageUrl}">) at "${path.join(topLvlRepoPath, pageSourcePath)}".  To resolve, change the value of the meta tag to be one of 🥀, 🔌, 🚪, or 🪟 and try running this script again.`);
                  }
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
        // Now build EJS partials from open positions in open-positions.yml. Note: We don't build these
        builtStaticContent.openPositions = [];// This will be passed into a component on the company handbook page to render a list of open positions.
        let RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO = 'handbook/company/open-positions.yml';

        // Get last modified timestamp using git, and represent it as a JS timestamp.
        let lastModifiedAt;
        if(!githubAccessToken) {
          lastModifiedAt = Date.now();
        } else {
          // If we're including a lastModifiedAt value for schema tables, we'll send a request to the GitHub API to get a timestamp of when the last commit
          let responseData = await sails.helpers.http.get.with({// [?]: https://docs.github.com/en/rest/commits/commits?apiVersion=2022-11-28#list-commits
            url: 'https://api.github.com/repos/fleetdm/fleet/commits',
            data: {
              path: RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO,
              page: 1,
              per_page: 1,//eslint-disable-line camelcase
            },
            headers: baseHeadersForGithubRequests,
          }).intercept((err)=>{
            return new Error(`When getting the commit history for the open positions YAML to get a lastModifiedAt timestamp, an error occured.`, err);
          });
          // The value we'll use for the lastModifiedAt timestamp will be date value of the `commiter` property of the `commit` we got in the API response from github.
          let mostRecentCommitToThisFile = responseData[0];
          if(!mostRecentCommitToThisFile.commit || !mostRecentCommitToThisFile.commit.committer) {
            // Throw an error if the the response from GitHub is missing a commit or commiter.
            throw new Error(`When trying to get a lastModifiedAt timestamp for the open positions YAML, the response from the GitHub API did not include information about the most recent commit. Response from GitHub: ${util.inspect(responseData, {depth:null})}`);
          }
          lastModifiedAt = (new Date(mostRecentCommitToThisFile.commit.committer.date)).getTime(); // Convert the UTC timestamp from GitHub to a JS timestamp.
        }

        let openPositionsYaml = await sails.helpers.fs.read(path.join(topLvlRepoPath, RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO)).intercept('doesNotExist', (err)=>new Error(`Could not find open positions YAML file at "${RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO}".  Was it accidentally moved?  Raw error: `+err.message));
        let openPositionsToCreatePartialsFor = YAML.parse(openPositionsYaml, {prettyErrors: true});
        // If the open positions yaml file is empty, the YAML parser will return it as null, and we can skip validating this file.
        if(openPositionsToCreatePartialsFor !== null) {
          for(let openPosition of openPositionsToCreatePartialsFor){
            // Make sure all open positions have the required values.
            if(!openPosition.jobTitle){
              throw new Error(`Error: could not build open position handbook pages from ${RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO}. An open position in the YAML is missing a "jobTitle". To resolve, add a "jobTitle" value and try running this script again.`);
            }

            if(!openPosition.department){
              throw new Error(`Error: could not build open position handbook pages from ${RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO}. An open position in the YAML is missing a "department" value. To resolve, add a "department" value to the "${openPosition.jobTitle}" position and try running this script again.`);
            }

            if(!openPosition.hiringManagerName){
              throw new Error(`Error: could not build open position handbook pages from ${RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO}. An open position in the YAML is missing a "hiringManagerName" value. To resolve, add a "hiringManagerName" value to the "${openPosition.jobTitle}" position and try running this script again.`);
            }

            if(!openPosition.hiringManagerLinkedInUrl){
              throw new Error(`Error: could not build open position handbook pages from ${RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO}. An open position in the YAML is missing a "hiringManagerLinkedInUrl" value. To resolve, add a "hiringManagerLinkedInUrl" value to the "${openPosition.jobTitle}" position and try running this script again.`);
            }

            if(!openPosition.hiringManagerGithubUsername){
              throw new Error(`Error: could not build open position handbook pages from ${RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO}. An open position in the YAML is missing a "hiringManagerGithubUsername" value. To resolve, add a "hiringManagerGithubUsername" value to the "${openPosition.jobTitle}" position and try running this script again.`);
            }

            if(!openPosition.responsibilities){
              throw new Error(`Error: could not build open position handbook pages from ${RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO}. An open position in the YAML is missing a "responsibilities" value. To resolve, add a "responsibilities" value to the "${openPosition.jobTitle}" position and try running this script again.`);
            }

            if(!openPosition.experience){
              throw new Error(`Error: could not build open position handbook pages from ${RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO}. An open position in the YAML is missing an "experience" value. To resolve, add an "experience" value to the "${openPosition.jobTitle}" position and try running this script again.`);
            }

            let pageTitle = openPosition.jobTitle;

            let mdStringForThisOpenPosition = `# ${openPosition.jobTitle}\n\n## Let's start with why we exist. 📡\n\nEver wondered if your employer is monitoring your work computer?\n\nOrganizations make huge investments every year to keep their laptops and servers online, secure, compliant, and usable from anywhere. This is called "device management".\n\nAt Fleet, we think it's time device management became [transparent](https://fleetdm.com/transparency) and [open source](https://fleetdm.com/handbook/company#open-source).\n\n\n## About the company 🌈\n\nYou can read more about the company in our [handbook](https://fleetdm.com/handbook/company), which is public and open to the world.\n\ntldr; Fleet Device Management Inc. is a [recently-funded](https://techcrunch.com/2022/04/28/fleet-nabs-20m-to-enable-enterprises-to-manage-their-devices/) Series A startup founded and backed by the same people who created osquery, the leading open source security agent. Today, osquery is installed on millions of laptops and servers, and it is especially popular with [enterprise IT and security teams](https://www.linuxfoundation.org/press/press-release/the-linux-foundation-announces-intent-to-form-new-foundation-to-support-osquery-community).\n\n\n## Your primary responsibilities 🔭\n${openPosition.responsibilities}\n\n## Are you our new team member? 🧑‍🚀\nIf most of these qualities sound like you, we would love to chat and see if we're a good fit.\n\n${openPosition.experience}\n\n## Why should you join us? 🛸\n\nLearn more about the company and [why you should join us here](https://fleetdm.com/handbook/company#is-it-any-good).\n\n<div purpose="open-position-quote-card"><div><img alt="Deloitte logo" src="/images/logo-deloitte-166x36@2x.png"></div><div purpose="open-position-quote"><div purpose="quote-text"><p>“One of the best teams out there to go work for and help shape security platforms.”</p></div><div purpose="quote-attribution"><strong>Dhruv Majumdar</strong><p>Director Of Cyber Risk & Advisory</p></div></div></div>\n\n\n## Want to join the team?\n\nWant to join the team?\n\nMessage us on [LinkedIn](https://www.linkedin.com/company/fleetdm/). \n\n\n >The salary range for this role is $48,000 - $480,000. Fleet provides competitive compensation based on our [compensation philosophy](https://fleetdm.com/handbook/company/communications#compensation), as well as comprehensive [benefits](https://fleetdm.com/handbook/company/communications#benefits).`;


            let htmlStringForThisPosition = await sails.helpers.strings.toHtml.with({mdString: mdStringForThisOpenPosition});

            // Modify links in the generated html string.
            htmlStringForThisPosition = htmlStringForThisPosition.replace(/(href="https?:\/\/([^"]+)")/g, (hrefString)=>{// « Modify links that are potentially external
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

            // Determine the htmlId for the generated page.
            let htmlId = (
              'handbook'+
              '--'+
              _.kebabCase(openPosition.jobTitle)+
              '--'+
              sails.helpers.strings.random.with({len:10})// if two files in different folders happen to have the same filename, there is a 1/16^10 chance of a collision (this is small enough- worst case, the build fails at the uniqueness check and we rerun it.)
            ).replace(/[^a-z0-9\-]/ig,'');

            // Determine the rootRelativeUrlPath for this open position, this will be used as the page's URL and to check if a markdown page already exists with this page's URL
            let rootRelativeUrlPath = '/handbook/company/open-positions/'+encodeURIComponent(_.kebabCase(openPosition.jobTitle));

            // If there is an existing page with the generated url, throw an error.
            if (rootRelativeUrlPathsSeen.includes(rootRelativeUrlPath)) {
              throw new Error(`Failed compiling markdown content: The "jobTitle" of an open position in ${RELATIVE_PATH_TO_OPEN_POSITIONS_YML_IN_FLEET_REPO} matches an existing page in the company handbook folder, and named would result in colliding (duplicate) URLs for the website.  To resolve, rename the pages whose names are too similar.  Duplicate detected: ${rootRelativeUrlPath}`);
            }//•

            // Generate ejs partial
            let htmlOutputPath = path.resolve(sails.config.appPath, path.join(APP_PATH_TO_COMPILED_PAGE_PARTIALS, htmlId+'.ejs'));
            if (dry) {
              sails.log('Dry run: Would have generated file:', htmlOutputPath);
            } else {
              await sails.helpers.fs.write(htmlOutputPath, htmlStringForThisPosition);
            }

            builtStaticContent.markdownPages.push({
              url: rootRelativeUrlPath,
              title: pageTitle,
              lastModifiedAt: lastModifiedAt,
              htmlId: htmlId,
              sectionRelativeRepoPath: 'company/open-positions.yml', // This is used to create the url for the "Edit this page" link
              meta: {maintainedBy: openPosition.hiringManagerGithubUsername},// Set the page maintainer to be the position's hiring manager.
            });
            // Add the positon to builtStaticContent.openPositions
            builtStaticContent.openPositions.push({
              jobTitle: openPosition.jobTitle,
              url: rootRelativeUrlPath,
            });
          }

        }
        // After we build the Markdown pages, we'll merge the osquery schema with the Fleet schema overrides, then create EJS partials for each table in the merged schema.
        let expandedTables;
        if(githubAccessToken){
          expandedTables = await sails.helpers.getExtendedOsquerySchema.with({includeLastModifiedAtValue: true, githubAccessToken,});
        } else {
          expandedTables = await sails.helpers.getExtendedOsquerySchema();
        }

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
            for(let column of _.sortBy(table.columns, 'name')) {
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
              if(column.hidden) { // If a column has `"hidden": true`, we'll add a note to the description that will be added to the table
                columnDescriptionForTable += '<br> **Not returned in `SELECT * FROM '+table.name+'`.**';
              }
              if(column.platforms) { // If a column has an array of platforms, we'll add a note to the final column description

                let platformString = '<br> **Only available on ';// start building a string to add to the column's description

                if(column.platforms.length > 3) {// FUTURE: add support for more than three platform values in columns.
                  throw new Error('Support for more than three platforms in columns has not been implemented yet. If this column is supported on all platforms, you can omit the platforms array entirely.');
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
              lastModifiedAt: table.lastModifiedAt,
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
        let RELATIVE_PATH_TO_PRICING_TABLE_YML_IN_FLEET_REPO = 'handbook/company/pricing-features-table.yml';
        let yaml = await sails.helpers.fs.read(path.join(topLvlRepoPath, RELATIVE_PATH_TO_PRICING_TABLE_YML_IN_FLEET_REPO)).intercept('doesNotExist', (err)=>new Error(`Could not find pricing table features YAML file at "${RELATIVE_PATH_TO_PRICING_TABLE_YML_IN_FLEET_REPO}".  Was it accidentally moved?  Raw error: `+err.message));
        let pricingTableFeatures = YAML.parse(yaml, {prettyErrors: true});
        let VALID_PRODUCT_CATEGORIES = ['Endpoint operations', 'Device management', 'Vulnerability management'];
        let VALID_PRICING_TABLE_CATEGORIES = ['Support', 'Deployment', 'Integrations', 'Configuration', 'Devices', 'Vulnerability management'];
        let VALID_PRICING_TABLE_KEYS = ['industryName', 'description', 'documentationUrl', 'tier', 'jamfProHasFeature', 'jamfProtectHasFeature', 'usualDepartment', 'productCategories', 'pricingTableCategories', 'waysToUse', 'buzzwords', 'demos', 'dri', 'friendlyName', 'moreInfoUrl', 'comingSoonOn', 'screenshotSrc', 'isExperimental'];
        for(let feature of pricingTableFeatures){
          // Throw an error if a feature contains an unrecognized key.
          for(let key of _.keys(feature)){
            if(!VALID_PRICING_TABLE_KEYS.includes(key)){
              throw new Error(`Unrecognized key. Could not build pricing table config from pricing-features-table.yml. The "${feature.industryName}" feature contains an unrecognized key (${key}). To resolve, fix any typos or remove this key and try running this script again.`);
            }
          }
          if(feature.name) {// Compatibility check
            throw new Error(`Could not build pricing table config from pricing-features-table.yml. A feature has a "name" (${feature.name}) which is no longer supported. To resolve, add a "industryName" to this feature: ${feature}`);
          }
          if(feature.industryName !== undefined) {
            if(!feature.industryName || typeof feature.industryName !== 'string') {
              throw new Error(`Could not build pricing table config from pricing-features-table.yml. A feature has a missing or invalid "industryName". To resolve, set an "industryName" as a valid, non-empty string for this feature ${feature}`);
            }
            feature.name = feature.industryName;//« This is just an alias. FUTURE: update code elsewhere to use the new property instead, and delete this aliasing.
          }
          if(!feature.productCategories){
            throw new Error(`Could not build pricing table config from pricing-features-table.yml. The '${feature.industryName}' feature is missing a 'productCategories' value. Please add an array of product categories to this feature and try running this script again`);
          } else {
            if(!_.isArray(feature.productCategories)){
              throw new Error(`Could not build pricing table config from pricing-features-table.yml. The '${feature.industryName}' feature has an invalid 'productCategories' value. Please change the productCategories for this feature to be an array of product categories`);
            } else {
              for(let category of feature.productCategories){
                if(!_.contains(VALID_PRODUCT_CATEGORIES, category)){
                  throw new Error(`Could not build pricing table config from pricing-features-table.yml. The '${feature.industryName}' feature has a 'productCategories' with an an invalid product category (${category}). Please change the values in this array to be one of: ${VALID_PRODUCT_CATEGORIES.join(', ')}`);
                }
              }
            }
          }
          if(!feature.pricingTableCategories){
            throw new Error(`Could not build pricing table config from pricing-features-table.yml. The ${feature.industryName} feature is missing a 'pricingTableCategory' value. Please add this value to this feature to be the category in the pricing table`);
          } else if(!_.isArray(feature.pricingTableCategories)){
            throw new Error(`Could not build pricing table config from pricing-features-table.yml. The ${feature.industryName} feature has an invalid 'pricingTableCategory' value. Please change the productCategories for this feature to be an array of pricing table categories. Type of invalid pricingTableCategory value: ${typeof feature.pricingTableCategory}`);
          } else {
            for(let category of feature.pricingTableCategories){
              if(!VALID_PRICING_TABLE_CATEGORIES.includes(category)){
                throw new Error(`Could not build pricing table config from pricing-features-table.yml. The ${feature.industryName} feature has an invalid 'pricingTableCategory' value. Please set this value to be one of: "${VALID_PRICING_TABLE_CATEGORIES.join('", "')}" and try running this script again. Invalid pricing table value: ${category}`);
              }
            }
          }
          if(!feature.tier) { // Throw an error if a feature is missing a `tier`.
            throw new Error(`Could not build pricing table config from pricing-features-table.yml. The ${feature.industryName} feature is missing a "tier". To resolve, add a "tier" (either "Free" or "Premium") to this feature.`);
          } else if(!_.contains(['Free', 'Premium'], feature.tier)){ // Throw an error if a feature's `tier` is not "Free" or "Premium".
            throw new Error(`Could not build pricing table config from pricing-features-table.yml. The ${feature.industryName} feature has an invalid "tier". to resolve, change the value of this features "tier" (currently set to '+feature.tier+') to be either "Free" or "Premium".`);
          }
          if(feature.comingSoon) {// Compatibility check
            throw new Error(`Could not build pricing table config from pricing-features-table.yml. A feature (industryName: ${feature.industryName}) category has "comingSoon", which is no longer supported. To resolve, remove "comingSoon" or add "comingSoonOn" (YYYY-MM-DD) to this feature ${feature}`);
          }
          if(feature.comingSoonOn !== undefined) {
            if(typeof feature.comingSoonOn !== 'string'){
              throw new Error(`Could not build pricing table config from pricing-features-table.yml. The ${feature.industryName} feature has an invalid "comingSoonOn" value (currently set to ${feature.comingSoonOn}, but expecting a string like 'YYYY-MM-DD'.)`);
            }
            feature.comingSoon = true;//« This is just an alias. FUTURE: update code elsewhere to use the new property instead, and delete this aliasing.
          }//ﬁ
        }
        builtStaticContent.pricingTable = pricingTableFeatures;
      },
      async()=>{
        // Validate the pricing table yaml and add it to builtStaticContent.pricingTable.
        let RELATIVE_PATH_TO_TESTIMONIALS_YML_IN_FLEET_REPO = 'handbook/company/testimonials.yml';
        let VALID_PRODUCT_CATEGORIES = ['Observability', 'Device management', 'Software management'];
        let yaml = await sails.helpers.fs.read(path.join(topLvlRepoPath, RELATIVE_PATH_TO_TESTIMONIALS_YML_IN_FLEET_REPO)).intercept('doesNotExist', (err)=>new Error(`Could not find testimonials YAML file at "${RELATIVE_PATH_TO_TESTIMONIALS_YML_IN_FLEET_REPO}".  Was it accidentally moved?  Raw error: `+err.message));
        let testimonials = YAML.parse(yaml, {prettyErrors: true});
        for(let testimonial of testimonials){
          // Throw an error if any value in the testimonial yaml is not a string.
          for(let key of _.keys(testimonial)) {
            if(typeof testimonial[key] !== 'string' && key !== 'productCategories'){
              throw new Error(`Could not build testimonial config from testimonials.yml. A testimonial contains a ${key} with a non-string value. Please make sure all values in testimonials.yml are strings, and try running this script again. Invalid (${typeof testimonial[key]}) ${key} value: ${testimonial[key]}`);
            }
          }
          // Check for required values.
          if(!testimonial.quote) {
            throw new Error(`Could not build testimonial config from testimonials.yml. A testimonial is missing a "quote". To resolve, make sure all testimonials have a "quote" and try running this script again. Testimonial missing a quote: ${testimonial}`);
          }
          if(!testimonial.quoteAuthorName) {
            throw new Error(`Could not build testimonial config from testimonials.yml. A testimonial is missing a "quoteAuthorName". To resolve, make sure all testimonials have a "quoteAuthorName", and try running this script again. Testimonial with missing "quoteAuthorName": ${testimonial} `);
          }
          if(!testimonial.quoteLinkUrl){
            throw new Error(`Could not build testimonial config from testimonials.yml. A testimonial is missing a "quoteLinkUrl" (A link to the quote author's LinkedIn profile). To resolve, make sure all testimonials have a "quoteLinkUrl", and try running this script again. Testimonial with missing "quoteLinkUrl": ${testimonial} `);
          }
          if(!testimonial.quoteAuthorProfileImageFilename){
            throw new Error(`Could not build testimonial config from testimonials.yml. A testimonial is missing a "quoteAuthorProfileImageFilename" (The quote author's LinkedIn profile picture). To resolve, make sure all testimonials have a "quoteAuthorProfileImageFilename", and try running this script again. Testimonial with missing "quoteAuthorProfileImageFilename": ${testimonial} `);
          } else {
            let imageFileExists = await sails.helpers.fs.exists(path.join(topLvlRepoPath, 'website/assets/images/', testimonial.quoteAuthorProfileImageFilename));
            if(!imageFileExists){
              throw new Error(`Could not build testimonials config from testimonials.yml. A testimonial has a 'quoteAuthorProfileImageFilename' value that points to an image that doesn't exist. Please make sure the file exists in the /website/assets/images/ folder. Invalid quoteImageFilename value: ${testimonial.quoteImageFilename}`);
            }
          }
          if(!testimonial.productCategories) {
            throw new Error(`Could not build testimonial config from testimonials.yml. A testimonial is missing a 'productCategories' value. Please add an array of product categories to this testimonial and try running this script again`);
          } else {
            if(!_.isArray(testimonial.productCategories)){
              throw new Error(`Could not build testimonial config from testimonials.yml. A testimonial has a has an invalid 'productCategories' value. Please change the productCategories for this testimonial to be an array of product categories`);
            } else {
              for(let category of testimonial.productCategories){
                if(!_.contains(VALID_PRODUCT_CATEGORIES, category)){
                  throw new Error(`Could not build testimonial config from testimonials.yml. A testimonial has a 'productCategories' with an an invalid product category (${category}). Please change the values in this array to be one of: ${VALID_PRODUCT_CATEGORIES.join(', ')}`);
                }
              }
            }
          }
          // If the testimonial has a youtubeVideoUrl, we'll validate the link and add the video ID so we can embed the video in a modal.
          if(testimonial.youtubeVideoUrl) {
            let videoLinkToCheck;
            try {
              videoLinkToCheck = new URL(testimonial.youtubeVideoUrl);
            } catch(err) {
              throw new Error(`Could not build testimonial config from testimonials.yml. When trying to parse a "youtubeVideoUrl" value, an erro occured. Please make sure all "youtubeVideoUrl" values are valid URLs and standard Youtube links (e.g, https://www.youtube.com/watch?v=siXy9aanOu4), and try running this script again. Invalid "youtubeVideoUrl" value: ${testimonial.youtubeVideoUrl}. error: ${err}`);
            }
            // If this is a youtu.be link, the video ID will be the pathname of the URL.
            if(!videoLinkToCheck.host.match(/w*\.*youtube\.com$/)) {
              throw new Error(`Could not build testimonials config from testimonials.yml. A testimonial has a "youtubeVideoUrl" that is a valid youtube link, but does not link to a video. Please make sure all "youtubeVideoLink" values are standard youtube links (e.g, https://www.youtube.com/watch?v=siXy9aanOu4) and try running this script again. invalid "youtubeVideoUrl" value: ${testimonial.youtubeVideoUrl}`);
            }
            // If this is a youtube.com link, the video ID will be in a query string.
            if(!videoLinkToCheck.search){
              // Throw an error if there is no video
              throw new Error(`Could not build testimonials config from testimonials.yml. A testimonial has a "youtubeVideoUrl" that is a valid youtube link, but does not link to a video. Please make sure all "youtubeVideoLink" values are standard youtube links (e.g, https://www.youtube.com/watch?v=siXy9aanOu4) and try running this script again. Invalid "youtubeVideoUrl" value: ${testimonial.youtubeVideoUrl}`);
            }
            let linkSearchParams = new URLSearchParams(videoLinkToCheck.search);
            if(!linkSearchParams.has('v')){
              throw new Error(`Could not build testimonials config from testimonials.yml. A testimonial has a "youtubeVideoUrl" that is a valid youtube link, but does not link to a video. Please make sure all "youtubeVideoLink" values are standard youtube links (e.g, https://www.youtube.com/watch?v=siXy9aanOu4) and try running this script again. Invalid "youtubeVideoUrl" value: ${testimonial.youtubeVideoUrl}`);
            }
            testimonial.videoIdForEmbed = linkSearchParams.get('v');
          }
          // Validate that all linked images exist, and that they match the website image name conventsions.
          // We'll also get the images dimensions from the filename, and add an imageHeight value to the testimonial.
          if(testimonial.quoteImageFilename) {
            // Throw an error if a testimonial with an image does not have a "quoteLinkUrl"
            if(!testimonial.quoteLinkUrl){
              throw new Error(`Could not build testimonial config from testimonials.yml. A testimonial with a 'quoteImageFilename' value is missing a 'quoteLinkUrl'. If providing a 'quoteImageFilename', a quoteLinkUrl (The link that the image will go to) is required. Testimonial missing a quoteLinkUrl: ${testimonial}`);
            }
            // Check if the image used for the testimonials exists.
            let imageFileExists = await sails.helpers.fs.exists(path.join(topLvlRepoPath, 'website/assets/images/', testimonial.quoteImageFilename));
            if(!imageFileExists){
              throw new Error(`Could not build testimonials config from testimonials.yml. A testimonial has a 'quoteImageFilename' value that points to an image that doesn't exist. Please make sure the file exists in the /website/assets/images/ folder. Invalid quoteImageFilename value: ${testimonial.quoteImageFilename}`);
            }
            let imageFilenameMatchesWebsiteConventions = testimonial.quoteImageFilename.match(/\d+x\d+@2x\.png|jpg|jpeg$/g);
            if(!imageFilenameMatchesWebsiteConventions) {
              throw new Error(`Could not build testimonials config from testimonials.yml. A testimonial has a quoteImageFilename that does not match the website\'s naming conventions. To resolve, make sure that the images dimensions are added to the filename, and that the filename ends with @2x. Filename that does not match the Fleet website's naming conventions: ${testimonial.quoteImageFilename}`);
            }
            // Strip the 2x from the filename, using image dimensions we matched when we checked if the filename matches website conventions.
            let extensionlessFilenameWithPostfixRemoved = imageFilenameMatchesWebsiteConventions[0].split('@2x')[0];
            // Get the height from the filename.
            let imagePathStringSections = extensionlessFilenameWithPostfixRemoved.split('x');
            let imageHeight = imagePathStringSections[imagePathStringSections.length - 1];
            testimonial.imageHeight = Number(imageHeight);
          }
        }
        builtStaticContent.testimonials = testimonials;
      },
      async()=>{
        let rituals = {};
        // Find all the files in the top level /handbook folder and it's sub-folders
        let FILES_IN_HANDBOOK_FOLDER = await sails.helpers.fs.ls.with({
          dir: path.join(topLvlRepoPath, '/handbook'),
          depth: 3
        });
        // Filter the list of filenames to get the rituals YAML files.
        let ritualTablesYamlFiles = FILES_IN_HANDBOOK_FOLDER.filter((filePath)=>{
          return _.endsWith(filePath, 'rituals.yml');
        });

        let githubLabelsToCheck = {};
        let KNOWN_AUTOMATABLE_FREQUENCIES = ['Daily', 'Weekly', 'Triweekly', 'Monthly', 'Annually'];
        // Process each rituals YAML file. These will be added to the builtStaticContent as JSON
        for(let ritualsYamlFilePath of ritualTablesYamlFiles){
          // Get this rituals.yml file's parent folder name, we'll use this as the key for this section's rituals in the ritualsTables dictionary
          let relativeRepoPathForThisRitualsFile = path.relative(topLvlRepoPath, ritualsYamlFilePath);
          // Parse the rituals YAML file.
          let yaml = await sails.helpers.fs.read(ritualsYamlFilePath);
          let ritualsFromRitualTableYaml = YAML.parse(yaml, {prettyErrors: true});

          // Make sure each ritual in the rituals YAML file has a task, startedOn, frequency, description, and DRI.
          for(let ritual of ritualsFromRitualTableYaml){
            if(!ritual.task){ // Throw an error if a ritual is missing a task
              throw new Error(`Could not build rituals from ${ritualsYamlFilePath}. A ritual in the YAML file is missing a task. To resolve add a task value (the name of the ritual) and try running this script again`);
            }
            if(!ritual.startedOn){// Throw an error if a ritual is missing a startedOn value.
              throw new Error(`Could not build rituals from ${ritualsYamlFilePath}. A ritual in the YAML file is missing a startedOn. To resolve add a startedOn value to the "${ritual.task}" ritual and try running this script again`);
            }
            if(!ritual.frequency){// Throw an error if a ritual is missing a frequency value.
              throw new Error(`Could not build rituals from ${ritualsYamlFilePath}. A ritual in the YAML file is missing a frequency. To resolve add a frequency value to the "${ritual.task}" ritual and try running this script again`);
            }
            if(!ritual.description){// Throw an error if a ritual is missing a description value.
              throw new Error(`Could not build rituals from ${ritualsYamlFilePath}. A ritual in the YAML file is missing a description. To resolve add a description value to the "${ritual.task}" ritual and try running this script again`);
            }
            if(!ritual.dri){// Throw an error if a ritual is missing a dri value.
              throw new Error(`Could not build rituals from ${ritualsYamlFilePath}. A ritual in the YAML file is missing a DRI. To resolve add a DRI value to the "${ritual.task}" ritual and try running this script again`);
            }
            if (ritual.autoIssue) { // If this ritual has an autoIssue value, we'll check to make sure the frequency is supported, and that the autoIssue value has a label value (an array of strings).
              if (!KNOWN_AUTOMATABLE_FREQUENCIES.includes(ritual.frequency)) {
                throw new Error(`Could not build rituals from ${ritualsYamlFilePath}. Invalid ritual: "${ritual.task}" indicates frequency "${ritual.frequency}", but that isn't supported with automations turned on.  Supported frequencies: ${KNOWN_AUTOMATABLE_FREQUENCIES}`);
              }
              if(githubAccessToken){ // If the ritual has an autoIssue value, we'll validate that the DRI value is a GitHub username.
                await sails.helpers.http.get.with({
                  url: 'https://api.github.com/users/' + encodeURIComponent(ritual.dri),
                  headers: baseHeadersForGithubRequests
                }).intercept((err)=>{
                  if(err.raw.statusCode === 404) {// If the GitHub API responds with a 404, we'll throw an error with a message about the invalid GitHub username.
                    return new Error(`Could not build rituals from ${ritualsYamlFilePath}. The DRI value of a ritual (${ritual.task}) contains an invalid GitHub username (${ritual.dri}). To resolve, make sure the DRI value for this ritual is a valid GitHub username.`);
                  } else {// If the error was not a 404, we'll display the full error
                    return err;
                  }
                });
              }
              if(!ritual.autoIssue.labels || !_.isArray(ritual.autoIssue.labels)){ // If the autoIssue value exists, but does not contain an array of labels, throw an error
                throw new Error(`Could not build rituals from ${ritualsYamlFilePath}. "${ritual.task}" contains an invalid autoIssue value. To resolve, add a "labels" value (An array of strings) to the autoIssue value.`);
              }
              if(!ritual.autoIssue.repo || typeof ritual.autoIssue.repo !== 'string') {
                throw new Error(`Could not build rituals from ${ritualsYamlFilePath}. "${ritual.task}" has an 'autoIssue' value that is missing a 'repo'. Please add the name of the repo that issues will be created in to the "autoIssue.repo" value and try running this script again.`);
              }
              if(!_.contains(['fleet', 'confidential'], ritual.autoIssue.repo)) {
                throw new Error(`Could not built rituals from ${ritualsYamlFilePath}. The "autoIssue.repo" value of "${ritual.task}" contains an invalid GitHub repo (${ritual.autoIssue.repo}). Please change this value to be either "fleet" or "confidential" and try running this script again.`);
              }
              // Check each label in the labels array
              for(let label of ritual.autoIssue.labels) {
                if(typeof label !== 'string') {
                  throw new Error(`Could not build rituals from ${ritualsYamlFilePath}. A ritual (${ritual.task}) in the YAML file contains an invalid value in the labels array of the autoIssue value. To resolve, ensure every value in the nested labels array of the autoIssue value is a string.`);
                }
                if(!githubLabelsToCheck[ritual.autoIssue.repo]){
                  // Create an empty array if an array does not exist for this repo.
                  githubLabelsToCheck[ritual.autoIssue.repo] = [];
                }
                // Add this label to the array of labels to check. We'll check to see if all labels are valid at the after we've processed all rituals YAML files.
                githubLabelsToCheck[ritual.autoIssue.repo].push({
                  label: label,
                  ritualUsingLabel: ritual.task,
                  ritualsYamlFilePath: relativeRepoPathForThisRitualsFile
                });
              }//∞
            }

          }
          // Add the rituals from this file to the rituals dictionary, using the file's relativeRepoPath (e.g, handbook/company/rituals.md) as a key.
          rituals[relativeRepoPathForThisRitualsFile] = ritualsFromRitualTableYaml;

        }//∞
        if(githubAccessToken) {
        // Validate all GitHub labels used in all ritual yaml files. Note: We check these here to minimize requests to the GitHub API.
          for(let repo in githubLabelsToCheck){
            let allExistingLabelsInSpecifiedRepo = [];
            let pageOfResultsReturned = 0;
            // Get all the labels in the specified repo.
            await sails.helpers.flow.until(async ()=>{
              let pageOfLabels = await sails.helpers.http.get.with({
                url: `https://api.github.com/repos/fleetdm/${repo}/labels?per_page=100&page=${pageOfResultsReturned}`,
                headers: baseHeadersForGithubRequests
              });
              allExistingLabelsInSpecifiedRepo = allExistingLabelsInSpecifiedRepo.concat(pageOfLabels);
              pageOfResultsReturned++;
              // This will stop running once all pages of labels in the specified GitHub repo have been returned.
              return pageOfLabels.length < 100;
            }, 10000);//∞   (maximum of 10s before giving up)
            // Get an array containing only the names of labels.
            let allLabelNamesInSpecifiedRepo = _.pluck(allExistingLabelsInSpecifiedRepo, 'name');
            // Validate each label, if a label does not exist in the specified repo, throw an error.
            await sails.helpers.flow.simultaneouslyForEach(githubLabelsToCheck[repo], async(labelInfo)=>{
              if(!_.contains(allLabelNamesInSpecifiedRepo, labelInfo.label)){
                throw new Error(`Could not build rituals from ${labelInfo.ritualsYamlFilePath}. The labels array nested within the autoIssue value of a ritual (${labelInfo.ritualUsingLabel}) contains an invalid GitHub label (${labelInfo.label}). To resolve, make sure all labels in the labels array are labels that exist in the repo that is soecificed in the .`);
              }
            });//∞
          }
        }//ﬁ

        // Add the rituals dictionary to builtStaticContent.rituals
        builtStaticContent.rituals = rituals;
      },
      //
      //   █████╗ ██████╗ ██████╗     ██╗     ██╗██████╗ ██████╗  █████╗ ██████╗ ██╗   ██╗
      //  ██╔══██╗██╔══██╗██╔══██╗    ██║     ██║██╔══██╗██╔══██╗██╔══██╗██╔══██╗╚██╗ ██╔╝
      //  ███████║██████╔╝██████╔╝    ██║     ██║██████╔╝██████╔╝███████║██████╔╝ ╚████╔╝
      //  ██╔══██║██╔═══╝ ██╔═══╝     ██║     ██║██╔══██╗██╔══██╗██╔══██║██╔══██╗  ╚██╔╝
      //  ██║  ██║██║     ██║         ███████╗██║██████╔╝██║  ██║██║  ██║██║  ██║   ██║
      //  ╚═╝  ╚═╝╚═╝     ╚═╝         ╚══════╝╚═╝╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝
      //
      async()=>{
        let appLibrary = [];
        // Get app library json
        let appsJsonData = await sails.helpers.fs.readJson(path.join(topLvlRepoPath, '/server/mdm/maintainedapps/apps.json'));
        // Then for each item in the json, build a configuration object to add to the sails.builtStaticContent.appLibrary array.
        await sails.helpers.flow.simultaneouslyForEach(appsJsonData, async(app)=>{
          let appInformation = {
            identifier: app.identifier,
            bundleIdentifier: app.bundle_identifier,
            installerFormat: app.installer_format,
          };
          // Note: This method of getting information about the apps will be out of date until the JSON files in the /server/mdm/maintainedapps/testdata/ folder are updated.
          let detailedInformationAboutThisApp = await sails.helpers.fs.readJson(path.join(topLvlRepoPath, '/server/mdm/maintainedapps/testdata/'+app.identifier+'.json'))
          .intercept('doesNotExist', ()=>{
            return new Error(`Could not build app library configuration from testdata folder. When attempting to read a JSON configuration file for ${app.identifier}, no file was found at ${path.join(topLvlRepoPath, '/server/mdm/maintainedapps/testdata/'+app.identifier+'.json. Was it moved?')}.`);
          });

          // Grab the latest information about these apps from the Homebrew API.
          // let detailedInformationAboutThisApp = await sails.helpers.http.get(`https://formulae.brew.sh/api/cask/${app.identifier}.json`)
          // .intercept((error)=>{
          //   return new Error(`Could not build app library configuration. When attempting to send a request to the homebrew API to get the latest information about ${app.identifier}, an error occured. Full error: ${util.inspect(error, {depth: null})}`);
          // });
          let scriptToUninstallThisApp = await sails.helpers.fs.read(path.join(topLvlRepoPath, `/server/mdm/maintainedapps/testdata/scripts/${app.identifier}_uninstall.golden.sh`))
          .intercept('doesNotExist', ()=>{
            return new Error(`Could not build app library configuration from testdata folder. When attempting to read an uninstall script for ${app.identifier}, no file was found at ${path.join(topLvlRepoPath, '/server/mdm/maintainedapps/testdata/scripts/'+app.identifier+'_uninstall.golden.sh. Was it moved?')}.`);
          });
          // Remove lines that only contain comments.
          scriptToUninstallThisApp = scriptToUninstallThisApp.replace(/^\s*#.*$/gm, '');

          // Condense functions onto a single line.
          // For each function in the script:
          scriptToUninstallThisApp = scriptToUninstallThisApp.replace(/(\w+)\s*\(\)\s*\{([\s\S]*?)^\}/gm, (match, functionName, functionContent)=> {
            // Split the function content into an array
            let linesInFunction = functionContent.split('\n');

            // Remove extra leading or trailing whitespace from each line.
            linesInFunction = linesInFunction.map((line)=>{ return line.trim();});

            // Remove any empty lines
            linesInFunction = linesInFunction.filter((lineText)=>{
              return lineText.length > 0;
            });
            // Iterate through the lines in the function, adding semicolons to lines with commands.
            linesInFunction = linesInFunction.map((text, lineIndex, lines)=>{
              // If this is not the last line in the function, and it does not only contain a control stucture keyword, append a semi colon to it.
              if(lineIndex !== lines.length - 1 && !/^\s*(if|while|for|do|else|then|done|return)/.test(text)) {
                return text + ';';
              }
              // Otherwise, do not add a semicolon
              return text;
            });
            // Join the lines into a single string
            let condensedBodyOfFunction = linesInFunction.join(' ');

            // Return the condensed single-line function.
            return `${functionName}() { ${condensedBodyOfFunction} }`;
          });

          // Remove newlines with "&&" and remove any that are added to the end and beginning of the condensed command.
          scriptToUninstallThisApp = scriptToUninstallThisApp.replace(/\n\s*/g, ' && ').replace(/ && $/, '').replace(/^ && /, '');


          appInformation.uninstallScript = scriptToUninstallThisApp;
          appInformation.version = detailedInformationAboutThisApp.version.split(',')[0];
          appInformation.description = detailedInformationAboutThisApp.desc;
          appInformation.name = detailedInformationAboutThisApp.name[0];
          appLibrary.push(appInformation);
        });
        builtStaticContent.appLibrary = appLibrary;
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

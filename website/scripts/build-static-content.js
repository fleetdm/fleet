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
      async()=>{// Parse query library from YAML and bake them into the Sails app's configuration.
        let RELATIVE_PATH_TO_QUERY_LIBRARY_YML_IN_FLEET_REPO = 'docs/1-Using-Fleet/standard-query-library/standard-query-library.yml';
        let yaml = await sails.helpers.fs.read(path.join(topLvlRepoPath, RELATIVE_PATH_TO_QUERY_LIBRARY_YML_IN_FLEET_REPO));

        let queriesWithProblematicRemediations = [];
        let queries = YAML.parseAllDocuments(yaml).map((yamlDocument)=>{
          let query = yamlDocument.toJSON().spec;
          query.slug = _.kebabCase(query.name);// « unique slug to use for routing to this query's detail page
          if ((query.remediation !== undefined && !_.isString(query.remediation)) || (query.purpose !== 'Detection' && _.isString(query.remediation))) {
            // console.log(typeof query.remediation);
            queriesWithProblematicRemediations.push(query);
          } else if (query.remediation === undefined) {
            query.remediation = 'N/A';// « We set this to a string here so that the data type is always string.  We use N/A so folks can see there's no remediation and contribute if desired.
          }
          return query;
        });
        // Report any errors that were detected along the way in one fell swoop to avoid endless resubmitting of PRs.
        if (queriesWithProblematicRemediations.length >= 1) {
          throw new Error('Failed parsing YAML for query library: The "remediation" of a query should either be absent (undefined) or a single string (not a list of strings).  And "remediation" should only be present when a query\'s purpose is "Detection".  But one or more queries have an invalid "remediation": ' + _.pluck(queriesWithProblematicRemediations, 'slug').sort());
        }//•
        // Assert uniqueness of slugs.
        if (queries.length !== _.uniq(_.pluck(queries, 'slug')).length) {
          throw new Error('Failed parsing YAML for query library: Queries as currently named would result in colliding (duplicate) slugs.  To resolve, rename the queries whose names are too similar.  Note the duplicates: ' + _.pluck(queries, 'slug').sort());
        }//•

        // Parse all queries to identify queries with problematic values for contributors.
        const queriesWithProblematicContributors = queries.reduce((list, query) => {
          if (!query.contributors || (query.contributors !== undefined && !_.isString(query.contributors))) {
            list.push(query);
            return list;
          } else if (query.contributors.split(',').some((contributor) => contributor.match('^[^A-za-z0-9].*|[^A-Za-z0-9-]|.*[^A-za-z0-9]$'))) {
            // GitHub Usernames may only contain alphanumeric characters or single hyphens, and cannot begin or end with a hyphen.
            list.push(query);
          }
          return list;
        }, []);
        // Report any errors that were detected along the way in one fell swoop to avoid endless resubmitting of PRs.
        if (queriesWithProblematicContributors.length >= 1) {
          // throw new Error('ERROR Failed parsing YAML for query library: The "contributors" of a query should either be absent (undefined) or a single string of valid GitHub user names (e.g. "zwass, noahtalerman, mikermcneil" rather than ["zwass", "noahtalerman", "mikermcneil"]).  But one or more queries have an invalid "contributors" value: ' + _.pluck(queriesWithProblematicContributors, 'slug').sort());
          console.log('WARNING: Failed parsing YAML for query library: The "contributors" of a query should either be absent (undefined) or a single string of valid GitHub user names (e.g. "zwass, noahtalerman, mikermcneil" rather than ["zwass", "noahtalerman", "mikermcneil"]).  But one or more queries have an invalid "contributors" value: ' + _.pluck(queriesWithProblematicContributors, 'slug').sort());
        }//•

        // Map all queries to build a list of unique contributor names then build a dictionary of user profile information from the GitHub Users API
        const contributorsList = queries.reduce((list, query) => {
          if (!queriesWithProblematicContributors.find((element) => element.slug === query.slug)) {
            list = _.union(list, query.contributors.split(','));
          }
          return list;
        }, []);

        const _threadGitHubAPICalls = async function (usersList) {

          // TODO replace with sails http helper
          const _getGitHubUserData = async (gitHubHandle) => {
            const url =
              'https://api.github.com/users/' + encodeURIComponent(gitHubHandle);
            const userData = await sails.helpers.http.get(url, {}, { 'User-Agent': 'Awesome-Octocat-App', Accept: 'application/vnd.github.v3+json' })
              .catch((error) => console.log('ERROR: ', error));
            return userData;
          };

          // Create threads object with a thread for each user. Each thread is a promise that will resolve
          // when the async call to the GitHub API resolves for that user.
          const threads =  _.union(usersList).reduce((threads, userName) => {
            threads[userName] = _getGitHubUserData(userName);
            return threads;
          }, {});

          // Each thread resolves with a key-value pair where the key is the user's GitHub login (aka '"handle"') and the value
          // is the deserialized JSON response returned by the GitHub API for that contributor.
          const resolvedThreads = await Promise.all(
            Object.keys(threads).map((key) =>
              Promise.resolve(threads[key]).then((result) => {
                return { [key]: result };
              })
            )
          ).then((resultsArray) => {
            return resultsArray.reduce((resolvedThreads, result) => {
              Object.assign(resolvedThreads, result);
              return resolvedThreads;
            }, {});
          });

          return resolvedThreads;

        };

        const contributorsDictionary = await _threadGitHubAPICalls(contributorsList);//•

        // Map all queries to replace the "contributors" string with an array of selected user profile information pulled from
        // the contributorsDictionary for each contributor (or undefined if the value is not a string although any problematic
        // queries should already have thrown an error above).
        queries = queries.map((query) => {
          if (!query.contributors || !_.isString(query.contributors)) {
            return query;
          }
          query.contributors = query.contributors.split(',').map((userName) => {
            return {
              userName,
              avatarUrl: contributorsDictionary[userName]['avatar_url'],
              htmlUrl: contributorsDictionary[userName]['html_url']
            };
          });
          return query;
        });//•
        // console.log(queries.map((q) => q.contributors));

        // TODO consider if/how to cache avatar images

        // Attach to Sails app configuration.
        builtStaticContent.queries = queries;
        builtStaticContent.queryLibraryYmlRepoPath = RELATIVE_PATH_TO_QUERY_LIBRARY_YML_IN_FLEET_REPO;
      },
      async()=>{// Parse markdown pages, compile & generate HTML files, and bake documentation's directory tree into the Sails app's configuration.

        // Note:
        // • path maths inspired by https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L107-L132
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        // // Original way that works:  (versus new stuff below)
        // builtStaticContent.markdownPages = await sails.helpers.compileMarkdownContent('docs/');  // TODO remove this and helper once everything works again
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

        builtStaticContent.markdownPages = [];// « dir tree representation that will be injected into Sails app's configuration

        let SECTION_REPO_PATHS = ['docs/', 'handbook/'];
        for (let sectionRepoPath of SECTION_REPO_PATHS) {
          let thinTree = await sails.helpers.fs.ls.with({
            dir: path.join(topLvlRepoPath, sectionRepoPath),
            depth: 100,
            includeDirs: false,
            includeSymlinks: false,
          });

          let rootRelativeUrlPathsSeen = [];
          for (let pageSourcePath of thinTree) {

            // Perform path maths (determine this using sectionRepoPath, etc)
            // > Inspired by https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L308-L313
            // > And https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L107-L132
            let rootRelativeUrlPath = `/todo-${_.trimRight(sectionRepoPath,'/')}-${pageSourcePath.slice(-30).replace(/[^a-z0-9\-]/ig,'')}-${Math.floor(Math.random()*10000000)}`;
            sails.log.verbose(`Building page ${rootRelativeUrlPath} from ${pageSourcePath} (${sectionRepoPath})`);
            // ^^TODO: replace that with the actual desired root relative URL path

            // Assert uniqueness of URL paths.
            if (rootRelativeUrlPathsSeen.includes(rootRelativeUrlPath)) {
              throw new Error('Failed compiling markdown content: Files as currently named would result in colliding (duplicate) URLs for the website.  To resolve, rename the pages whose names are too similar.  Duplicate detected: ' + rootRelativeUrlPath);
            }//•
            rootRelativeUrlPathsSeen.push(rootRelativeUrlPath);

            // Get last modified timestamp using git
            // > Inspired by https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L265-L273
            let lastModifiedAt = Date.now();// TODO

            let fallbackTitle = sails.helpers.strings.toSentenceCase(path.basename(pageSourcePath, '.ejs'));// « for clarity (the page isn't a template, necessarily, and this title is just a guess.  Display title will, more likely than not, come from a <docmeta> tag -- see the bottom of the original, raw unformatted markdown of any page in the sailsjs docs for an example of how to use docmeta tags)

            // If markdown: Compile to HTML and parse docpage metadata
            // > Parsing docmeta tags (consider renaming them to just <meta>- or by now there's probably a more standard way of embedding semantics in markdown files; prefer to use that): https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L180-L183
            // > Compiling: https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L198-L202
            // TODO

            // Skip this page, if appropriate
            // > Inspired by https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L275-L276
            // TODO

            // Generate HTML file
            let htmlOutputPath = '';//TODO
            if (dry) {
              sails.log('Dry run: Would have generated file:', htmlOutputPath);
            } else {
              // TODO
            }

            // TODO: Figure out what to do about embedded images (they'll get cached by CDN so probably ok to point at github, but markdown img srcs will break if relative.  Also GitHub could just change image URLs whenever.)

            // Append to Sails app configuration.
            builtStaticContent.markdownPages.push({
              url: rootRelativeUrlPath,
              title: '' || fallbackTitle,// TODO use metadata title if available
              lastModifiedAt: lastModifiedAt
            });
          }//∞ </each source file>
        }//∞ </each section repo path>

        // Decorate markdownPages tree with easier-to-use properties related to metadata embedded in the markdown and parent/child relationships.
        // Note: Maybe skip the parent/child relationships.
        // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/helpers/marshal-doc-page-metadata.js
        // > And https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/build-jsmenu.js
        // TODO

        // Sort siblings in the markdownPages tree so it's ready to use in menus.
        // > Note: consider doing this on the frontend-- though there's a reason it was here. See:
        // > • https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/helpers/compare-doc-page-metadatas.js
        // > • https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/helpers/marshal-doc-page-metadata.js#L191-L208
        // TODO
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

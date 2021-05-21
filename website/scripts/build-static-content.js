module.exports = {


  friendlyName: 'Build static content',


  description: 'Generate HTML partials from source files in fleetdm/fleet repo (e.g. docs in markdown, or queries in YAML), and configure metadata about the generated files so it is available in `sails.config.builtStaticContent`.',


  inputs: {
    dry: { type: 'boolean', description: 'Whether to make this a dry run.  (.sailsrc file will not be overwritten.  HTML files will not be generated.)' },
  },


  fn: async function ({ dry }) {

    let path = require('path');
    let YAML = require('yaml');

    // Rather than involving git, we'll just use the current repo to get the source files we need.
    // (See git history for examples of another strategy if we need source files from other places.)
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    // The data we're compiling will get built into this dictionary and then written on top of the .sailsrc file.
    let builtStaticContent = {};

    await sails.helpers.flow.simultaneously([
      async()=>{// Parse query library from YAML and bake them into the Sails app's configuration.
        let yaml = await sails.helpers.fs.read(path.join(topLvlRepoPath, 'docs/1-Using-Fleet/standard-query-library/standard-query-library.yml'));
        let queries = YAML.parseAllDocuments(yaml).map((yamlDocument)=>{
          let query = yamlDocument.toJSON().spec;
          query.slug = _.kebabCase(query.name);// « unique slug to use for routing to this query's detail page
          return query;
        });
        // Assert uniqueness of slugs.
        if (queries.length !== _.uniq(_.pluck(queries, 'slug')).length) {
          throw new Error('Failed parsing YAML for query library: Queries as currently named would result in colliding (duplicate) slugs.  To resolve, rename the queries whose names are too similar.  Note the duplicates: ' + _.pluck(queries, 'slug').sort());
        }//•
        // Attach to Sails app configuration.
        builtStaticContent.queries = queries;
      },
      async()=>{// Parse markdown pages, compile & generate HTML files, and bake documentation's directory tree into the Sails app's configuration.

        // Note:
        // • path maths inspired by inspired by https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L107-L132
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        // // Original way that works:  (versus new stuff below)
        // builtStaticContent.allPages = await sails.helpers.compileMarkdownContent('docs/');  // TODO remove this and helper once everything works again
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

        builtStaticContent.allPages = [];// « dir tree representation that will be injected into Sails app's configuration

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
            // > See https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L308-L313
            // > And https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L107-L132
            let rootRelativeUrlPath = '/'+Math.floor(Math.random()*10000000);// TODO

            // Assert uniqueness of URL paths.
            if (rootRelativeUrlPathsSeen.includes(rootRelativeUrlPath)) {
              throw new Error('Failed compiling markdown content: Files as currently named would result in colliding (duplicate) URLs for the website.  To resolve, rename the pages whose names are too similar.  Duplicate detected: ' + rootRelativeUrlPath);
            }//•
            rootRelativeUrlPathsSeen.push(rootRelativeUrlPath);

            // Get last modified timestamp using git
            // > See https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L265-L273
            let lastModifiedAt = Date.now();// TODO

            let fallbackTitle = sails.helpers.strings.toSentenceCase(path.basename(pageSourcePath, '.ejs'));// « for clarity (the page isn't a template, necessarily, and this title is just a guess.  Display title will, more likely than not, come from a <docmeta> tag -- see the bottom of the original, raw unformatted markdown of any page in the sailsjs docs for an example of how to use docmeta tags)

            // If markdown: Compile to HTML and parse docpage metadata
            // > Parsing docmeta tags (consider renaming them to just <meta>- or by now there's probably a more standard way of embedding semantics in markdown files; prefer to use that): https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L180-L183
            // > Compiling: https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L198-L202
            // TODO

            // Skip this page, if appropriate
            // > See https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L275-L276
            // TODO

            // Generate HTML file
            let htmlOutputPath = '';//TODO
            if (dry) {
              sails.log('Dry run: Would have generated file:', htmlOutputPath);
            } else {
              // TODO
            }

            // FUTURE: Figure out what to do about embedded images (they'll get cached by CDN so probably ok to point at github, but markdown img srcs will break if relative.  Also GitHub could just change image URLs whenever.)

            // Append to Sails app configuration.
            builtStaticContent.allPages.push({
              url: rootRelativeUrlPath,
              title: '' || fallbackTitle,// TODO use metadata title if available
              lastModifiedAt: lastModifiedAt
            });
          }//∞ </each source file>
        }//∞ </each section repo path>

        // Decorate allPages tree with easier-to-use properties related to metadata embedded in the markdown and parent/child relationships.
        // Note: Maybe skip the parent/child relationships.
        // > See https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/helpers/marshal-doc-page-metadata.js
        // > And https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/build-jsmenu.js
        // TODO

        // Sort siblings in the allPages tree so it's ready to use in menus.
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

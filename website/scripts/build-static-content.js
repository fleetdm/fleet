module.exports = {


  friendlyName: 'Build static content',


  description: 'Generate HTML partials from source files in fleetdm/fleet repo (e.g. docs in markdown, or queries in YAML), and configure metadata about the generated files so it is available in `sails.config.builtStaticContent`.',


  inputs: {
    dry: { type: 'boolean', description: 'Whether to make this a dry run.  (.sailsrc file will not be overwritten.  Everything else still happens).' },
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
        builtStaticContent.queries = YAML.parseAllDocuments(yaml).map((yamlDocument) => yamlDocument.toJSON().spec );
      },
      async()=>{// Parse documentation, compile HTML/sitemap.xml, and bake documentation's directory tree into the Sails app's configuration.
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        // // Original way that works:  (versus new stuff below)
        // builtStaticContent.documentationTree = await sails.helpers.compileMarkdownContent('docs/');
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

        let thinTree = await sails.helpers.fs.ls.with({
          dir: path.join(topLvlRepoPath, 'docs/'),
          depth: 100,
          includeDirs: false,
          includeSymlinks: false,
        });

        // Build directory tree to be injected into Sails app's configuration.
        let thickTree = thinTree.map((pageSourcePath)=>{
          return {
            path: pageSourcePath,// TODO remove this prbly for clarity
            url: pageSourcePath,// TODO: root relative URL path
            fallbackTitle: sails.helpers.strings.toSentenceCase(path.basename(pageSourcePath, '.ejs')),// « for clarity (the page isn't a template, necessarily, and this title is just a guess.  Display title will, more likely than not, come from a <docmeta> tag -- see the bottom of the original, raw unformatted markdown of any page in the sailsjs docs for an example of how to use docmeta tags)
            lastModifiedAt: Date.now()// «TODO
          };
        });
        builtStaticContent.documentationTree = thickTree;

        // Loop over doc pages building sitemap.xml data and generating HTML files.
        // > • path maths inspired by inspired by https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L107-L132
        // > • sitemap building inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/refresh.js#L112-L180 and https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/helpers/get-pages-for-sitemap.js
        // > • Why escape XML?  See http://stackoverflow.com/questions/3431280/validation-problem-entityref-expecting-what-should-i-do and https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/refresh.js#L161-L172
        let sitemapXml = '<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">';
        [// Start with root relative URLs of other webpages that aren't being generated from markdown
          '/',
          '/get-started',
          // TODO rest  (e.g. hand-coded HTML pages from routes.js -- see https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/helpers/get-pages-for-sitemap.js#L27)
        ].forEach((url)=>{
          sitemapXml += `<url><loc>${_.escape(`https://fleetdm.com${url}`)}</loc></url>`;// note we omit lastmod. This is ok, to mix w/ other entries that do have lastmod. Why? See https://docs.google.com/document/d/1SbpSlyZVXWXVA_xRTaYbgs3750jn252oXyMFLEQxMeU/edit
        });
        for (let pageInfo of thickTree) {
          // TODO: If markdown: Compile to HTML and parse docpage metadata
          // TODO: Skip this page, if appropriate
          // TODO: Perform path maths
          // TODO: Generate HTML file
          sitemapXml +=`<url><loc>${_.escape(`https://fleetdm.com${pageInfo.url}`)}</loc><lastmod>${pageInfo.lastModifiedAt}</lastmod></url>`;
        }//∞

        // Generate sitemap.xml file
        sitemapXml += '</urlset>';
        console.log(sitemapXml);
        // TODO
        // TODO: make sure this gets checked in in GH actions workflow

      },
    ]);

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

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
        builtStaticContent.documentationTree = await sails.helpers.compileMarkdownContent('docs/');
        // TODO: Generate XML sitemap (see "refresh" action in sailsjs.com repo for example)
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


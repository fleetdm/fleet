module.exports = {


  friendlyName: 'Build static content',


  description: 'Generate HTML partials from source files in fleetdm/fleet repo (e.g. docs in markdown, or queries in YAML), and configure metadata about the generated files so it is available in `sails.config.builtStaticContent`.',


  inputs: {
    dry: { type: 'boolean', description: 'Whether to make this a dry run.  (.sailsrc file will not be overwritten.  Everything else still happens).' },
  },


  fn: async function ({ dry }) {

    let path = require('path');
    let YAML = require('yaml');

    // The data we're compiling will get built into this dictionary and then written on top of the .sailsrc file.
    let builtStaticContent = {};

    // Compile queries from YAML to markdown.
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // FUTURE: Be smarter (faster) about how we get and compile this YAML.
    // i.e. dissect some of the code from here https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L16-L22
    // and use those building blocks directly instead of depending on doctemplater later and thus
    // unnecessarily duplicating work.   i.e. the "clone repo" step can be pulled up ahead of the simultaneously, etc.
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // Clone repo
    let topLvlCachePath = path.resolve(sails.config.paths.tmp, `built-static-content/`);
    await sails.helpers.fs.rmrf(topLvlCachePath);
    let repoCachePath = path.join(topLvlCachePath, `cloned-repo-${Date.now()}-${Math.round(Math.random()*100)}`);
    await sails.helpers.process.executeCommand(`git clone git://github.com/fleetdm/fleet.git ${repoCachePath}`);

    // Parse YAML query library
    let yaml = await sails.helpers.fs.read(path.join(repoCachePath, 'docs/1-Using-Fleet/standard-query-library/standard-query-library.yml'));
    builtStaticContent.queries = YAML.parseAllDocuments(yaml).map((yamlDocument) => yamlDocument.toJSON().spec );

    // Legacy:  (TODO: remove this later on, once in-progress frontend work is done)
    await sails.helpers.compileMarkdownContent('handbook/queries/'); // TODO: pair w/ rachel and swap how page is built now that this part will work differently


    // Compile HTML from markdown.
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // FUTURE: Make this work in parallel as shown here by improving doctemplater to avoid the alreadyExists error
    // (this actually only fails the very first time, but still, thinking is that it's not worth leaving a hack in
    // here for a trivial build perf boost right now, especially since it only affects website deploys)
    // ```
    // let builtStaticContent = await sails.helpers.flow.simultaneously({
    //   documentation: async() => await sails.helpers.compileMarkdownContent('docs/'),
    //   queryLibrary: async() => await sails.helpers.compileMarkdownContent('handbook/queries/')
    // });
    // console.log(builtStaticContent);
    // ```
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    builtStaticContent.documentationTree = await sails.helpers.compileMarkdownContent('docs/');


    // Compile and generate XML sitemap
    // (see "refresh" action in sailsjs.com repo for example)
    // TODO


    // Replace .sailsrc file.
    // > This takes the compiled menu file from doc-templater and injects it into the .sailsrc file so it
    // > can be accessed for the purposes of config using `sails.config.builtStaticContent`.
    if (dry) {
      console.log('Dry run: Would have folded the following onto .sailsrc as "builtStaticContent":', builtStaticContent);
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


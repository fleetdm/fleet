module.exports = {


  friendlyName: 'Build static content',


  description: 'Generate HTML partials from source files in fleetdm/fleet repo (e.g. docs in markdown, or queries in YAML), and configure metadata about the generated files so it is available in `sails.config.builtStaticContent`.',


  fn: async function () {

    let path = require('path');

    // Compile queries from YAML to markdown.
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // FUTURE: Be smarter about how we compile the YAML.
    // i.e. dissect some of the code from here https://github.com/uncletammy/doc-templater/blob/2969726b598b39aa78648c5379e4d9503b65685e/lib/compile-markdown-tree-from-remote-git-repo.js#L16-L22
    // and use those building blocks directly instead of depending on doctemplater
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // TODO

    // Compile HTML from markdown.
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // FUTURE: Make this work in parallel as shown here by improving doctemplater to avoid the alreadyExists error
    // (this actually only fails the very first time, but still, thinking is that it's not worth leaving a hack in
    // here for a trivial build perf boost right now, especially since it only affects website deploys)
    // ```
    // let filesGeneratedBySection = await sails.helpers.flow.simultaneously({
    //   documentation: async() => await sails.helpers.compileMarkdownContent('docs/'),
    //   queryLibrary: async() => await sails.helpers.compileMarkdownContent('handbook/queries/')
    // });
    // console.log(filesGeneratedBySection);
    // ```
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    let filesGeneratedBySection = {
      documentation: await sails.helpers.compileMarkdownContent('docs/'),
      queryLibrary: await sails.helpers.compileMarkdownContent('handbook/queries/')
    };

    // Compile and generate XML sitemap
    // (see "refresh" action in sailsjs.com repo for example)
    // TODO

    // Replace .sailsrc file.
    // > This takes the compiled menu file from doc-templater and injects it into the .sailsrc file so it
    // > can be accessed for the purposes of config using `sails.config.builtStaticContent`.
    let sailsrcPath = path.resolve(sails.config.appPath, '.sailsrc');
    let oldSailsrcJson = await sails.helpers.fs.readJson(sailsrcPath);
    await sails.helpers.fs.writeJson.with({
      force: true,
      destination: sailsrcPath,
      json: {
        ...oldSailsrcJson,
        builtStaticContent: filesGeneratedBySection,
      }
    });

  }


};


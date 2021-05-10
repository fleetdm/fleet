module.exports = {


  friendlyName: 'Build from markdown content',


  description: '',


  fn: async function () {

    let path = require('path');

    // sails.log('Running custom shell script... (`sails run compile-markdown-content`)');

    let filesGeneratedBySection = {
      documentation: await sails.helpers.compileMarkdownContent('docs/'),
      queryLibrary: await sails.helpers.compileMarkdownContent('handbook/queries/')
    };
    // FUTURE: Make this work in parallel as shown here by improving doctemplater to avoid the alreadyExists error
    // (this actually only fails the very first time, but still, thinking is that it's not worth leaving a hack in
    // here for a trivial build perf boost right now, especially since it only affects website deploys)
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // let filesGeneratedBySection = await sails.helpers.flow.simultaneously({
    //   documentation: async() => await sails.helpers.compileMarkdownContent('docs/'),
    //   queryLibrary: async() => await sails.helpers.compileMarkdownContent('handbook/queries/')
    // });
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // console.log(filesGeneratedBySection);

    // Compile and generate XML sitemap
    // (see "refresh" action in sailsjs.com repo for example)
    // TODO

    // Now take the compiled menu file and inject it into the .sailsrc file so it
    // can be accessed for the purposes of config using `sails.config.builtFromMarkdown`.
    let sailsrcPath = path.resolve(sails.config.appPath, '.sailsrc');
    let sailsrcContent = await sails.helpers.fs.readJson(sailsrcPath);
    sailsrcContent.builtFromMarkdown = filesGeneratedBySection;
    await sails.helpers.fs.writeJson(sailsrcPath, sailsrcContent, true);

  }


};


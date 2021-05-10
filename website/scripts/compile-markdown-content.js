module.exports = {


  friendlyName: 'Compile markdown content',


  description: '',


  fn: async function () {

    let path = require('path');

    // sails.log('Running custom shell script... (`sails run compile-markdown-content`)');

    let filesGeneratedBySection = await sails.helpers.flow.simultaneously({
      documentation: async() => await sails.helpers.compileMarkdownContent('docs/'),
      queryLibrary: async() => await sails.helpers.compileMarkdownContent('handbook/queries/')
    });
    // console.log(filesGeneratedBySection);

    // Now take the compiled menu file and inject it into the .sailsrc file so it
    // can be accessed for the purposes of config using `sails.config.compiledFromMarkdown`.
    let sailsrcPath = path.resolve(sails.config.appPath, '.sailsrc');
    let sailsrcContent = await sails.helpers.fs.readJson(sailsrcPath);
    sailsrcContent.compiledFromMarkdown = filesGeneratedBySection;
    await sails.helpers.fs.writeJson(sailsrcPath, sailsrcContent, true);

  }


};


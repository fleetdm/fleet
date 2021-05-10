module.exports = {


  friendlyName: 'Compile markdown content',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run compile-markdown-content`)');

    let generatedFilesBySection = await sails.helpers.flow.simultaneously({
      documentation: async() => await sails.helpers.compileMarkdownContent('docs/'),
      queryLibrary: async() => await sails.helpers.compileMarkdownContent('handbook/queries/')
    });

    // Now take the compiled menu file and inject it into the .sailsrc file so it
    // can be accessed for the purposes of config.
    // TODO
    console.log(generatedFilesBySection);

  }


};


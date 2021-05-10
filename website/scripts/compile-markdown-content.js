module.exports = {


  friendlyName: 'Compile markdown content',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run compile-markdown-content`)');

    await sails.helpers.flow.simultaneously([
      async()=>{
        await sails.helpers.compileMarkdownContent('docs/');
      },
      async()=>{
        await sails.helpers.compileMarkdownContent('handbook/queries/');
      },
    ]);

  }


};


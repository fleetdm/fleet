module.exports = {


  friendlyName: 'Test ai improve prompt',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run test-ai-improve-prompt`)');

    let prompt1 = (
      'Given some data and a set of possible choices, decide which choice most accurately classifies the data.  Data: ```\n'+
      JSON.stringify([
        {
          id: 1,
          author: 'mikermcneil',
          tweet: 'I fed this one stray cat and now I have 20 stray cats coming to my house',
        },
        {
          id: 2,
          author: 'fancydoilies',
          tweet: 'My cat is named Rory'
        },
        {
          id: 3,
          author: 'koo',
          tweet: 'Sails.js is the best JavaScript framework'
        },
      ])+
      '```\n'+'\n'+'Choices:\n'+
      ' • A social media post that is both (a) VERY interesting and (b) in reasonably good taste\n'+
      ' • Anything else\n'+
      '\n'+
      'Decide based on which choice is the most correct for the given data.  Respond only with the exact string value for the choice provided.'
    );
    sails.log(await ƒ.improvePrompt(prompt1));

    sails.log(await ƒ.improvePrompt(prompt1),2);

    sails.log(await ƒ.improvePrompt(prompt1),5);

  }


};


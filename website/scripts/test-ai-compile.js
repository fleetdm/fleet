module.exports = {


  friendlyName: 'Test ai compile',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run test-ai-compile`)');

    // ƒ.compile('Create a user in the database and set the `.userId` key in the session.');
    return await ƒ.prompt.with({
      baseModel: 'o4-mini-2025-04-16',
      prompt:
        'Given a set of helper definitions, generate code for a sails app action (actions2 in sails v1) to accomplish the following specification:\n```\n'+
        'Sign up: Handle a signup form by creating a user in the database and set the `.userId` key in the session.'+
        '\n```\nHere are the helper defintions:\n```\n'+
        require('util').inspect(sails.helpers)+
        '\n```\nRespond only with JavaScript code.',
    });

  }


};


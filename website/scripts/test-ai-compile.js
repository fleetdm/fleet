module.exports = {


  friendlyName: 'Test ai compile',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run test-ai-compile`)');

    let goal1 = 'Fibonnacci: Respond with an array of a fibonacci sequence';
    sails.log(await ƒ.compile(goal1, 'helper'));

    let goal2 = 'Sign up: Handle a signup form.';
    sails.log(await ƒ.compile(goal2));

  }


};


module.exports = {


  friendlyName: 'Test ai compile',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run test-ai-compile`)');

    let goal1 = 'Fibonnacci: Respond with an array of a fibonacci sequence';
    sails.log(await ƒ.compile(goal1, 'helper'));

    let goal2 = 'Sign up: Handle a signup form.';
    sails.log(await ƒ.compile(goal2));

    let goal3 = 'Receive from Fleet: Handle a webhook sent by Fleet whenever a policy fails, such that, if the policy is critical, we send an email to the person\'s email.  Reach out to the Fleet API as needed to map the incoming data\'s hostname to the human email identity using the originating host.';
    sails.log(await sails.helpers.ai.compile(goal3));

  }


};


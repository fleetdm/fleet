module.exports = {


  friendlyName: 'Get new api token',


  description: '',


  inputs: {

  },


  exits: {

  },


  fn: async function (inputs) {
    let userToRegerateTokenFor = this.req.me;

    let newApiToken = await sails.helpers.strings.uuid();

    await User.updateOne({id:this.req.me.id}).set({apiToken: newApiToken});
    // All done.
    return newApiToken;

  }


};

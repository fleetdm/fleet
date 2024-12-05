module.exports = {


  friendlyName: 'Get new api token',


  description: 'Regenerates a Users API token and updates their database record.',


  inputs: {

  },


  exits: {
    success: {
      description: 'A new API token has been generated for a user.'
    }
  },


  fn: async function () {
    let newApiToken = await sails.helpers.strings.uuid();

    await User.updateOne({id:this.req.me.id}).set({apiToken: newApiToken});
    // All done.
    return newApiToken;

  }


};

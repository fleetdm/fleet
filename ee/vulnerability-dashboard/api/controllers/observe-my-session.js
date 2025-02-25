module.exports = {


  friendlyName: 'Observe my session',


  description: 'Subscribe to the logged-in user\'s session so that you receive socket broadcasts when logged out in another tab.',


  exits: {

    success: {
      description: 'The requesting socket is now subscribed to socket broadcasts about the logged-in user\'s session.',
    },

  },


  fn: async function ({}) {

    if (!this.req.isSocket) {
      throw new Error('This action is designed for use with the virtual request interpreter (over sockets, not traditional HTTP).');
    }

    let roomName = `session${_.deburr(this.req.sessionID)}`;
    sails.sockets.join(this.req, roomName);


  }


};

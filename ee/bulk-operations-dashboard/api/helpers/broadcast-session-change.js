module.exports = {


  friendlyName: 'Broadcast session change',


  description: 'Broadcast a socket notification indicating a change in login status.',


  inputs: {

    req: {
      type: 'ref',
      required: true,
    },

  },


  exits: {

    success: {
      description: 'All done.',
    },

  },


  fn: async function ({ req }) {

    // If there's no sessionID, we don't need to broadcase a message about the old session.
    if(!req.sessionID) {
      return;
    }

    let roomName = `session${_.deburr(req.sessionID)}`;
    let messageText = `You have signed out or signed into a different session in another tab or window. Reload the page to refresh your session.`;
    sails.sockets.broadcast(roomName, 'session', { notificationText: messageText }, req);


  }


};


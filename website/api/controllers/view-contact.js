module.exports = {


  friendlyName: 'View contact',


  description: 'Display "Contact" page.',

  inputs: {
    sendMessage: {
      type: 'boolean',
      description: 'A boolean that determines whether or not to display the talk to us form when the contact page loads.',
      defaultsTo: false,
    },
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/contact'
    }

  },


  fn: async function ({sendMessage}) {

    let formToShow = 'talk-to-us';

    if(sendMessage) {
      formToShow = 'contact';
    }
    // Respond with view.
    return {
      formToShow,
    };

  }


};

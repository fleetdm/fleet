module.exports = {


  friendlyName: 'View contact',


  description: 'Display "Contact" page.',

  inputs: {
    sendmessage: {
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


  fn: async function ({sendmessage}) {

    let formToDisplay = 'talk-to-us';

    if(sendmessage) {
      formToDisplay = 'contact';
    }
    // Respond with view.
    return {formToDisplay};

  }


};

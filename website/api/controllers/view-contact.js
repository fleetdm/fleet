module.exports = {


  friendlyName: 'View contact',


  description: 'Display "Contact" page.',

  inputs: {
    sendMessage: {
      type: 'boolean',
      description: 'A boolean that determines whether or not to display the talk to us form when the contact page loads.',
      defaultsTo: false,
    },

    prefillFormDataFromUserRecord: {
      type: 'boolean',
      description: 'If true, the contact form will be prefilled in with information from this user\'s account.',
    },
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/contact'
    }

  },


  fn: async function ({sendMessage, prefillFormDataFromUserRecord}) {

    let formToShow = 'talk-to-us';

    if(sendMessage) {
      formToShow = 'contact';
    }
    // If the prefillFormDataFromUserRecord flag was set to true, but this user is not logged in, set it to false.
    if(prefillFormDataFromUserRecord && !this.req.me){
      prefillFormDataFromUserRecord = false;
    }
    // Respond with view.
    return {formToShow, prefillFormDataFromUserRecord};

  }


};

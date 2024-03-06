module.exports = {


  friendlyName: 'View contact',


  description: 'Display "Contact" page.',

  inputs: {
    talkToUs: {
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


  fn: async function ({talkToUs}) {

    let formToDisplay = 'contact';

    if(talkToUs) {
      formToDisplay = 'talk-to-us';
    }
    // Respond with view.
    return {formToDisplay};

  }


};

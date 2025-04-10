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

    let userIsLoggedIn = !! this.req.me;
    let userHasPremiumSubscription = false;
    if(userIsLoggedIn) {
      let thisSubscription = await Subscription.findOne({user: this.req.me.id});
      if(thisSubscription){
        formToShow = 'contact';
        userHasPremiumSubscription = true;
      }
    }

    if(sendMessage) {
      formToShow = 'contact';
    }
    // Respond with view.
    return {
      formToShow,
      userIsLoggedIn,
      userHasPremiumSubscription
    };

  }


};

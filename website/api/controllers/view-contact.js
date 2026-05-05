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
    },

    badConfig: {
      responseType: 'badConfig'
    },

  },


  fn: async function ({sendMessage}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.openPositions)) {
      throw {badConfig: 'builtStaticContent.openPositions'};
    }

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

    let currentOpenPositionsForApplicationDropdown = _.pluck(sails.config.builtStaticContent.openPositions, 'jobTitle');


    // Respond with view.
    return {
      formToShow,
      userIsLoggedIn,
      userHasPremiumSubscription,
      currentOpenPositionsForApplicationDropdown
    };

  }


};

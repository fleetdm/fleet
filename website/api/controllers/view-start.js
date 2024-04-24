module.exports = {


  friendlyName: 'View start',


  description: 'Display "Start" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/start'
    },
    redirect: {
      description: 'The requesting user already has a Fleet Premium subscription.',
      responseType: 'redirect',
    }
  },


  fn: async function () {
    // If the user has a license key, we'll redirect them to the customer dashboard.
    let userHasExistingSubscription = await Subscription.findOne({user: this.req.me.id});
    if (userHasExistingSubscription) {
      throw {redirect: '/customers/dashboard'};
    }
    if(this.req.me.isSuperAdmin){
      throw {redirect: '/admin/generate-license'};
    }
    if(this.req.me.lastSubmittedGetStartedQuestionnaireStep && !_.isEmpty(this.req.me.getStartedQuestionnaireAnswers)){
      let currentStep = this.req.me.lastSubmittedGetStartedQuestionnaireStep;
      let previouslyAnsweredQuestions = this.req.me.getStartedQuestionnaireAnswers;
      // Respond with view.
      return {currentStep, previouslyAnsweredQuestions};
    } else {
      // Respond with view.
      return;
    }

  }


};

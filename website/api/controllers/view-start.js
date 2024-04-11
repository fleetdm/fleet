module.exports = {


  friendlyName: 'View start',


  description: 'Display "Start" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/start'
    }

  },


  fn: async function () {
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

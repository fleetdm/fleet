module.exports = {


  friendlyName: 'View start',


  description: 'Display "Start" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/start'
    }

  },


  fn: async function () {
    if(this.req.me.currentGetStartedQuestionnarieStep && this.req.me.getStartedQuestionnarieAnswers){
      let currentStep = this.req.me.currentGetStartedQuestionnarieStep;
      let previouslyAnsweredQuestions = this.req.me.getStartedQuestionnarieAnswers;
      // Respond with view.
      return {currentStep, previouslyAnsweredQuestions};
    } else {
      // Respond with view.
      return;
    }

  }


};

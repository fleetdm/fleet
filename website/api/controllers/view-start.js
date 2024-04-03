module.exports = {


  friendlyName: 'View start',


  description: 'Display "Start" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/start'
    }

  },


  fn: async function () {
    let currentStep = 'start';
    let previouslyAnsweredQuestions;
    if(this.req.me.currentGetStartedQuestionnarieStep && this.req.me.getStartedQuestionnarieAnswers){
      currentStep = this.req.me.currentGetStartedQuestionnarieStep;
      previouslyAnsweredQuestions = this.req.me.getStartedQuestionnarieAnswers;
    }
    // Respond with view.
    return {currentStep, previouslyAnsweredQuestions};

  }


};

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
    if(this.req.session.getStartedProgress && this.req.session.getStartedProgress.currentStep){
      currentStep = this.req.session.getStartedProgress.currentStep;
      previouslyAnsweredQuestions = this.req.session.getStartedProgress.previouslyAnsweredQuestions;
    }
    console.log(this.req.session.getStartedProgress);
    // Respond with view.
    return {currentStep, previouslyAnsweredQuestions};

  }


};

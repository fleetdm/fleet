module.exports = {


  friendlyName: 'Save questionnaire progress and continue',


  description: 'Saves the user\'s current progress in the get started questionnaire',


  inputs: {
    currentStep: {
      type: 'string',
      // isIn: [TODO]
    },
    formData: {
      type: {},
    }
  },


  exits: {

  },


  fn: async function ({currentStep, formData}) {
    let userRecord = await User.findOne({id: this.req.me.id});
    let questionnaireProgress;
    // console.log(userRecord.getStartedQuestionnarieAnswers);
    if(!userRecord.currentGetStartedQuestionnarieStep || _.isEmpty(userRecord.getStartedQuestionnarieAnswers)) {
      questionnaireProgress = {};
    } else {
      questionnaireProgress = _.clone(userRecord.getStartedQuestionnarieAnswers);
    }
    questionnaireProgress[currentStep] = formData;
    let getStartedProgress = _.clone(questionnaireProgress);
    await User.updateOne({id: userRecord.id}).set({getStartedQuestionnarieAnswers: questionnaireProgress, currentGetStartedQuestionnarieStep: currentStep});
    return getStartedProgress;
  }


};

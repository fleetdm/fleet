module.exports = {


  friendlyName: 'Save questionnaire progress and continue',


  description: 'Saves the user\'s current progress in the get started questionnaire',


  inputs: {
    currentStep: {
      type: 'string',
      description: 'The step of the get started questionnaire that is being saved.'
    },
    formData: {
      type: {},
      description: 'The formdata that will be saved for this step of the get started questionnaire'
    }
  },


  exits: {

  },


  fn: async function ({currentStep, formData}) {
    // find this user's DB record.
    let userRecord = await User.findOne({id: this.req.me.id});
    let questionnaireProgress;
    // If this user doesn't have a currentGetStartedQuestionnarieStep or getStartedQuestionnarieAnswers
    if(!userRecord.currentGetStartedQuestionnarieStep || _.isEmpty(userRecord.getStartedQuestionnarieAnswers)) {
      questionnaireProgress = {};
    } else {// other wise clone it from the user record.
      questionnaireProgress = _.clone(userRecord.getStartedQuestionnarieAnswers);
    }
    if(currentStep === 'what-are-you-using-fleet-for') {
      let primaryBuyingSituation = formData.primaryBuyingSituation;
      await User.updateOne({id: this.req.me.id}).set({primaryBuyingSituation});
    }
    // Set the user's answer to the current step.
    questionnaireProgress[currentStep] = formData;
    // Clone the questionnaireProgress to prevent any mutations from sending it through the updateOne Waterline method.
    let getStartedProgress = _.clone(questionnaireProgress);
    // Update the user's database model.
    await User.updateOne({id: userRecord.id}).set({getStartedQuestionnarieAnswers: questionnaireProgress, currentGetStartedQuestionnarieStep: currentStep});
    // Return the JSON dictionary of form data submitted by this user.
    return getStartedProgress;
  }


};

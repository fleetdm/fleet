module.exports = {


  friendlyName: 'Save questionnaire progress and continue',


  description: 'Saves the user\'s current progress in the get started questionnaire',


  inputs: {
    currentStep: {
      type: 'string',
      description: 'The step of the get started questionnaire that is being saved.',
      isIn: [
        'start',
        'what-are-you-using-fleet-for',
        'have-you-ever-used-fleet',
        'how-many-hosts',
        'will-you-be-self-hosting',
        'what-are-you-working-on-eo-security',
        'is-it-any-good',
        'what-did-you-think',
        'deploy-fleet-in-your-environment',
        'managed-cloud-for-growing-deployments',
        'self-hosted-deploy',
      ]
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
    if(!userRecord){
      throw new Error(`Consistency violation: when trying to save a user's progress in the get started questionnaire, a User record with the ID ${this.req.me.id} could not be found.`);
    }
    let questionnaireProgress;
    // If this user doesn't have a lastSubmittedGetStartedQuestionnaireStep or getStartedQuestionnaireAnswers, create an empty dictionary to store their answers.
    if(!userRecord.lastSubmittedGetStartedQuestionnaireStep || _.isEmpty(userRecord.getStartedQuestionnaireAnswers)) {
      questionnaireProgress = {};
    } else {// other wise clone it from the user record.
      questionnaireProgress = _.clone(userRecord.getStartedQuestionnaireAnswers);
    }
    // When the 'what-are-you-using-fleet-for' is completed, update this user's DB record and session to include their answer.
    if(currentStep === 'what-are-you-using-fleet-for') {
      let primaryBuyingSituation = formData.primaryBuyingSituation;
      await User.updateOne({id: this.req.me.id}).set({primaryBuyingSituation});
      // Set the primary buying situation in the user's session.
      this.req.session.primaryBuyingSituation = primaryBuyingSituation;
    }
    // Set the user's answer to the current step.
    questionnaireProgress[currentStep] = formData;
    // Clone the questionnaireProgress to prevent any mutations from sending it through the updateOne Waterline method.
    let getStartedProgress = _.clone(questionnaireProgress);
    // Update the user's database model.
    await User.updateOne({id: userRecord.id}).set({getStartedQuestionnaireAnswers: questionnaireProgress, lastSubmittedGetStartedQuestionnaireStep: currentStep});
    // Return the JSON dictionary of form data submitted by this user.
    return getStartedProgress;
  }


};

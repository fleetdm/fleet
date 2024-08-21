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
        'what-does-your-team-manage-eo-it',
        'what-does-your-team-manage-vm',
        'what-do-you-manage-mdm',
        'cross-platform-mdm',
        'is-it-any-good',
        'what-did-you-think',
        'deploy-fleet-in-your-environment',
        'managed-cloud-for-growing-deployments',
        'self-hosted-deploy',
        'whats-left-to-get-you-set-up',
        'how-was-your-deployment',
        'thanks-for-checking-out-fleet',
      ]
    },
    formData: {
      type: {},
      description: 'The formdata that will be saved for this step of the get started questionnaire'
    },
  },


  exits: {
    success: {
      outputDescription: 'All get started questionnaire answers accumulated so far by this user.',
      outputType: {}
    },
  },


  fn: async function ({currentStep, formData}) {
    // find this user's DB record.
    let userRecord = this.req.me;
    let questionnaireProgress;
    // If this user doesn't have a lastSubmittedGetStartedQuestionnaireStep or getStartedQuestionnaireAnswers, create an empty dictionary to store their answers.
    if(!userRecord.lastSubmittedGetStartedQuestionnaireStep || _.isEmpty(userRecord.getStartedQuestionnaireAnswers)) {
      questionnaireProgress = {};
    } else {// other wise clone it from the user record.
      questionnaireProgress = _.clone(userRecord.getStartedQuestionnaireAnswers);
    }

    // Tease out what liur buying situation will now be (or is and was, if it's not changing)
    let primaryBuyingSituation = formData.primaryBuyingSituation === undefined ? this.req.me.primaryBuyingSituation : formData.primaryBuyingSituation;

    // When the 'what-are-you-using-fleet-for' is completed, update this user's DB record and session to include their answer.
    if(currentStep === 'what-are-you-using-fleet-for') {
      await User.updateOne({id: this.req.me.id})
      .set({
        primaryBuyingSituation: primaryBuyingSituation
      });
      // Set the primary buying situation in the user's session.
      this.req.session.primaryBuyingSituation = primaryBuyingSituation;
    }//ﬁ

    //  ┌─┐┌─┐┌┬┐  ┌─┐┌─┐┬ ┬┌─┐┬ ┬┌─┐┬  ┌─┐┌─┐┬┌─┐┌─┐┬    ┌─┐┌┬┐┌─┐┌─┐┌─┐
    //  └─┐├┤  │   ├─┘└─┐└┬┘│  ├─┤│ ││  │ ││ ┬││  ├─┤│    └─┐ │ ├─┤│ ┬├┤
    //  └─┘└─┘ ┴   ┴  └─┘ ┴ └─┘┴ ┴└─┘┴─┘└─┘└─┘┴└─┘┴ ┴┴─┘  └─┘ ┴ ┴ ┴└─┘└─┘
    // This is how the questionnaire steps/options change a user's psychologicalStage value.
    // 'start': No change
    // 'what-are-you-using-fleet-for':
    //  - (any option) = stage 2
    // 'have-you-ever-used-fleet':
    //  - yes-deployed: » Stage 6
    //  - yes-recently-deployed: » Stage 5
    //  - yes-deployed-local: » Stage 3 (Tried Fleet but might not have a use case)
    //  - yes-deployed-long-time: Stage 2 (Tried Fleet long ago but might not fully grasp)
    //  - no: Stage 2 (Never tried Fleet and might not fully grasp)
    // 'how-many-hosts': Stage 4/5/6
    // 'will-you-be-self-hosting': Stage 5/6
    // 'what-are-you-working-on-eo-security'
    //  - All other options » Stage 4
    // 'what-does-your-team-manage-eo-it'
    //  - All other options » Stage 4
    // 'what-does-your-team-manage-vm'
    //  - All other options » Stage 4
    // 'what-do-you-manage-mdm'
    //  - no-use-case-yet: » Stage 3
    //  - All other options » Stage 4
    // 'is-it-any-good': Stage 3/4 (depends on answer from 'have-you-ever-used-fleet' & the buying situation specific step)
    // 'what-did-you-think'
    //  - host-fleet-for-me » Stage 5
    //  - deploy-fleet-in-environment » Stage 5
    //  - let-me-think-about-it »  Stage 2
    // FUTURE: Should the step about deploying fleet in your env be here?  (For same reason is-it-any-good is here: when navigating back then forwards?)
    // 'how-was-your-deployment'
    //  - up-and-running »  Stage 5
    //  - kinda-stuck »  Stage 5
    //  - havent-gotten-to-it » Stage 5
    //  - changed-mind-want-managed-deployment » Stage 5
    //  - decided-to-not-use-fleet » Stage 2
    // 'whats-left-to-get-you-set-up'
    //  - need-premium-license-key » No change (Stage ??)
    //  - help-show-fleet-to-my-team » No change (Stage ??)
    //  - procurement-wants-some-stuff » No change (Stage ??)
    //  - nothing » No change (Stage ??)

    let psychologicalStage = userRecord.psychologicalStage;
    let psychologicalStageLastChangedAt = userRecord.psychologicalStageLastChangedAt;
    // Get the value of the submitted formData, we do this so we only need to check one variable, instead of (formData.attribute === 'foo');
    let valueFromFormData = _.values(formData)[0];
    if(currentStep === 'start') {
      // There is change when the user completes the start step.
    } else if(currentStep === 'what-are-you-using-fleet-for') {
      psychologicalStage = '2 - Aware';
    } else if(currentStep === 'have-you-ever-used-fleet') {
      if(valueFromFormData === 'yes-deployed') {
        // If the user has Fleet deployed, set their stage to 6.
        psychologicalStage = '6 - Has team buy-in';
      } else if(valueFromFormData === 'yes-recently-deployed') {
        psychologicalStage = '5 - Personally confident';
      } else {
        psychologicalStage = '3 - Intrigued';
      }
    } else {
      // If the user submitted any other step, we'll set variables using the answers to the previous questions.
      // Get the user's selected primaryBuyingSiutation.
      let currentSelectedBuyingSituation = questionnaireProgress['what-are-you-using-fleet-for'].primaryBuyingSituation;
      // Get the user's answer to the "Have you ever used Fleet?" question.
      let hasUsedFleetAnswer = questionnaireProgress['have-you-ever-used-fleet'].fleetUseStatus;
      if(['what-are-you-working-on-eo-security','what-does-your-team-manage-eo-it','what-does-your-team-manage-vm','what-do-you-manage-mdm'].includes(currentStep)){
        if(valueFromFormData === 'no-use-case-yet') {
          psychologicalStage = '3 - Intrigued';
        } else {// Otherwise, they have a use case and will be set to stage 4.
          psychologicalStage = '4 - Has use case';
        }
      } else if(currentStep === 'is-it-any-good') {
        if(currentSelectedBuyingSituation === 'mdm') {
          // Since the mdm use case question is the only buying situation-specific question where a use case can't
          // be selected,  we'll check the user's previous answers before changing their psyStage
          if(questionnaireProgress['what-do-you-manage-mdm'].mdmUseCase === 'no-use-case-yet'){
            // Check the user's answer to the have-you-ever-used-fleet question.
            psychologicalStage = '3 - Intrigued';
          } else {
            psychologicalStage = '4 - Has use case';
          }
        } else {// For any other selected primary buying situation, since a use case will have been selected, set their psyStage to 4
          psychologicalStage = '4 - Has use case';
          // FUTURE: check previous answers for other selected buying situations.
        }
      } else if(currentStep === 'what-did-you-think') {// (what did you think about [presumably after you actually did...] trying it locally)
        // If the user selects "Let me think about it", set their psyStage to 2.
        if(valueFromFormData === 'let-me-think-about-it') {
          psychologicalStage = '2 - Aware';
        } else if (['host-fleet-for-me', 'deploy-fleet-in-environment'].includes(valueFromFormData)) {
          psychologicalStage = '5 - Personally confident';
        } else { require('assert')(false,'This should never happen.'); }
      } else if(currentStep === 'how-was-your-deployment') {
        if(valueFromFormData === 'decided-to-not-use-fleet') {
          psychologicalStage = '2 - Aware';
        } else if(['up-and-running', 'changed-mind-want-managed-deployment', 'kinda-stuck', 'havent-gotten-to-it'].includes(valueFromFormData)){
          psychologicalStage = '5 - Personally confident';
        } else { require('assert')(false,'This should never happen.'); }
      } else if (currentStep === 'whats-left-to-get-you-set-up') {
        // FUTURE: do more stuff (for now this always acts like 'no change')
      } else if(currentStep === 'how-many-hosts') {
        if(['yes-deployed'].includes(hasUsedFleetAnswer)) {
          psychologicalStage = '6 - Has team buy-in';
        } else {
          psychologicalStage = '5 - Personally confident';
        }
      } else if(currentStep === 'will-you-be-self-hosting') {
        if(['yes-deployed'].includes(hasUsedFleetAnswer)) {
          psychologicalStage = '6 - Has team buy-in';
        } else if(['yes-recently-deployed'].includes(hasUsedFleetAnswer)){
          psychologicalStage = '5 - Personally confident';
        } else { require('assert')(false, 'This should never happen.'); }
      } else if(currentStep === 'thanks-for-checking-out-fleet') {
        psychologicalStage = '2 - Aware';
      }//ﬁ
    }//ﬁ

    // Only update CRM records if the user's psychological stage changes.
    if(psychologicalStage !== userRecord.psychologicalStage) {
      // Update the psychologicalStageLastChangedAt timestamp if the user's psychological stage
      psychologicalStageLastChangedAt = Date.now();
      sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
        emailAddress: this.req.me.emailAddress,
        firstName: this.req.me.firstName,
        lastName: this.req.me.lastName,
        primaryBuyingSituation: primaryBuyingSituation === 'eo-security' ? 'Endpoint operations - Security' : primaryBuyingSituation === 'eo-it' ? 'Endpoint operations - IT' : primaryBuyingSituation === 'mdm' ? 'Device management (MDM)' : primaryBuyingSituation === 'vm' ? 'Vulnerability management' : undefined,
        organization: this.req.me.organization,
        psychologicalStage,
        contactSource: 'Website - Sign up',
      }).exec((err)=>{
        if(err){
          sails.log.warn(`Background task failed: When a user (email: ${this.req.me.emailAddress} submitted a step of the get started questionnaire, a Contact and Account record could not be created/updated in the CRM.`, err);
        }
        return;
      });
    }//ﬁ
    // TODO: send all other answers to Salesforce (when there are fields for them)
    // Set the user's answer to the current step.
    questionnaireProgress[currentStep] = formData;
    // Clone the questionnaireProgress to prevent any mutations from sending it through the updateOne Waterline method.
    let getStartedProgress = _.clone(questionnaireProgress);
    // Update the user's database model.
    await User.updateOne({id: userRecord.id})
    .set({
      getStartedQuestionnaireAnswers: questionnaireProgress,
      lastSubmittedGetStartedQuestionnaireStep: currentStep,
      psychologicalStage,
      psychologicalStageLastChangedAt,
    });
    // Return the JSON dictionary of form data submitted by this user.
    return {getStartedProgress, psychologicalStage};
  }


};

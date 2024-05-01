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
    //  - yes-recently-deployed: » Stage 6
    //  - yes-deployed-local: » Stage 3 (Tried Fleet but might not have a use case)
    //  - yes-deployed-long-time: Stage 2 (Tried Fleet long ago but might not fully grasp)
    //  - no: Stage 2 (Never tried Fleet and might not fully grasp)
    // 'how-many-hosts': Stage 6
    // 'will-you-be-self-hosting': Stage 6
    // 'what-are-you-working-on-eo-security'
    //  - no-use-case-yet: » Stage 2/3 (depends on answer from 'have-you-ever-used-fleet' step)
    //  - All other options » Stage 4
    // 'what-does-your-team-manage-eo-it'
    //  - no-use-case-yet: » Stage 2/3 (depends on answer from 'have-you-ever-used-fleet' step)
    //  - All other options » Stage 4
    // 'what-does-your-team-manage-vm'
    //  - no-use-case-yet: » Stage 2/3 (depends on answer from 'have-you-ever-used-fleet' step)
    //  - All other options » Stage 4
    // 'what-do-you-manage-mdm'
    //  - no-use-case-yet: » Stage 2/3 (depends on answer from 'have-you-ever-used-fleet' step)
    //  - All other options » Stage 4
    // 'is-it-any-good': Stage 2/3/4 (depends on answer from 'have-you-ever-used-fleet' & the buying situation specific step)
    // 'what-did-you-think'
    //  - deploy-fleet-in-environment » Stage 5
    //  - let-me-think-about-it »  Stage 2
    //  - host-fleet-for-me » N/A (currently not selectable, but should set the user's psychologicalStage to stage 5)

    let psychologicalStage = userRecord.psychologicalStage;
    // Get the value of the submitted formData, we do this so we only need to check one variable, instead of (formData.attribute === 'foo');
    let valueFromFormData = _.values(formData)[0];
    if(currentStep === 'start') {
      // There is change when the user completes the start step.
    } else if(currentStep === 'what-are-you-using-fleet-for') {
      psychologicalStage = '2 - Aware';
    } else if(currentStep === 'have-you-ever-used-fleet') {
      if(['yes-deployed', 'yes-recently-deployed'].includes(valueFromFormData)) {
        // If the user has Fleet deployed, set their stage to 6.
        psychologicalStage = '6 - Has team buy-in';
      } else if(valueFromFormData === 'yes-deployed-local'){
        // If they've tried Fleet locally, set their stage to 3.
        psychologicalStage = '3 - Intrigued';
      } else {
        // Otherwise, we'll just assume liu're only aware.  Maybe liu don't fully grasp what Fleet can do.
        psychologicalStage = '2 - Aware';
      }
    } else {
      // If the user submitted any other step, we'll set variables using the answers to the previous questions.
      // Get the user's selected primaryBuyingSiutation.
      let currentSelectedBuyingSituation = questionnaireProgress['what-are-you-using-fleet-for'].primaryBuyingSituation;
      // Get the user's answer to the "Have you ever used Fleet?" question.
      let hasUsedFleetAnswer = questionnaireProgress['have-you-ever-used-fleet'].fleetUseStatus;
      if(['what-are-you-working-on-eo-security','what-does-your-team-manage-eo-it','what-does-your-team-manage-vm','what-do-you-manage-mdm'].includes(currentStep)){
        if(valueFromFormData === 'no-use-case-yet') {
          // Check the user's answer to the previous question
          if(hasUsedFleetAnswer === 'yes-deployed-local'){
            // If they've tried Fleet locally, set their stage to 3.
            psychologicalStage = '3 - Intrigued';
          } else {
            psychologicalStage = '2 - Aware';
          }
        } else {// Otherwise, they have a use case and will be set to stage 4.
          psychologicalStage = '4 - Has use case';
        }
      } else if(currentStep === 'is-it-any-good') {
        if(currentSelectedBuyingSituation === 'mdm') {
          // Since the mdm use case question is the only buying situation-sepcific question where a use case can't
          // be selected,  we'll check the user's previous answers befroe changing their psyStage
          if(questionnaireProgress['what-do-you-manage-mdm'].mdmUseCase === 'no-use-case-yet'){
            // Check the user's answer to the have-you-ever-used-fleet question.
            if(hasUsedFleetAnswer === 'yes-deployed-local') {
              // If they've tried Fleet locally, set their stage to 3.
              psychologicalStage = '3 - Intrigued';
            } else {
              psychologicalStage = '2 - Aware';
            }
          } else {
            psychologicalStage = '4 - Has use case';
          }
        } else {// For any other selected primary buying situation, since a use case will have been selected, set their psyStage to 4
          psychologicalStage = '4 - Has use case';
          // FUTURE: check previous answers for other selected buying situations.
        }
      } else if(currentStep === 'what-did-you-think') {
        // If the user is ready to deploy Fleet in their work environemnt, then they're ready to get buy-in from their team, so set their psyStage to 5.
        if(valueFromFormData === 'deploy-fleet-in-environment') {
          psychologicalStage = '5 - Personally confident';
        } else if(valueFromFormData === 'let-me-think-about-it') {
        // If the user selects "Let me think about it", their stage change to 2
          psychologicalStage = '2 - Aware';
        }
        // If the user selects "I’d like you to host Fleet for me", the form is not submitted, and they are taken to the /contact page instead. FUTURE: set stage to stage 5.
      } else if(currentStep === 'how-many-hosts') {
        // If they have Fleet deployed, they have team buy-in
        psychologicalStage = '6 - Has team buy-in';
      } else if(currentStep === 'will-you-be-self-hosting') {
        // If they have Fleet deployed, they have team buy-in
        psychologicalStage = '6 - Has team buy-in';
      }//ﬁ

    }//ﬁ

    // Only update CRM records if the user's psychological stage changes.
    if(psychologicalStage !== userRecord.psychologicalStage){
      await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
        emailAddress: this.req.me.emailAddress,
        firstName: this.req.me.firstName,
        lastName: this.req.me.lastName,
        primaryBuyingSituation: primaryBuyingSituation === 'eo-security' ? 'Endpoint operations - Security' : primaryBuyingSituation === 'eo-it' ? 'Endpoint operations - IT' : primaryBuyingSituation === 'mdm' ? 'Device management (MDM)' : primaryBuyingSituation === 'vm' ? 'Vulnerability management' : undefined,
        organization: this.req.me.organization,
        psychologicalStage,
      });
    }
    // TODO: send all other answers to Salesforce (when there are fields for them)

    // await sails.helpers.http.post.with({
    //   url: 'https://hooks.zapier.com/hooks/catch/3627242/3nltwbg/',
    //   data: {
    //     emailAddress: this.req.me.emailAddress,
    //     firstName: this.req.me.firstName,
    //     lastName: this.req.me.lastName,
    //     primaryBuyingSituation: primaryBuyingSituation,
    //     organization: this.req.me.organization,
    //     psychologicalStage,
    //     currentStep,
    //     webhookSecret: sails.config.custom.zapierSandboxWebhookSecret,
    //   }
    // })
    // .timeout(5000)
    // .tolerate(['non200Response', 'requestFailed'], (err)=>{
    //   // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
    //   sails.log.warn(`When a user completed a questionnaire step, a lead/contact could not be updated in the CRM for this email address: ${this.req.me.emailAddress}. Raw error: ${err}`);
    //   return;
    // });
    // Set the user's answer to the current step.
    questionnaireProgress[currentStep] = formData;
    // Clone the questionnaireProgress to prevent any mutations from sending it through the updateOne Waterline method.
    let getStartedProgress = _.clone(questionnaireProgress);
    // Update the user's database model.
    await User.updateOne({id: userRecord.id})
    .set({
      getStartedQuestionnaireAnswers: questionnaireProgress,
      lastSubmittedGetStartedQuestionnaireStep: currentStep,
      psychologicalStage
    });
    // Return the JSON dictionary of form data submitted by this user.
    return getStartedProgress;
  }


};

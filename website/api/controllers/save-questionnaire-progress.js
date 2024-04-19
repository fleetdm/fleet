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
    // When the 'what-are-you-using-fleet-for' is completed, update this user's DB record and session to include their answer.
    if(currentStep === 'what-are-you-using-fleet-for') {
      let primaryBuyingSituation = formData.primaryBuyingSituation;
      await User.updateOne({id: this.req.me.id})
      .set({
        primaryBuyingSituation
      });
      // Send a POST request to Zapier
      await sails.helpers.http.post.with({
        url: 'https://hooks.zapier.com/hooks/catch/3627242/3pl7yt1/',
        data: {
          primaryBuyingSituation,
          emailAddress: this.req.me.emailAddress,
          webhookSecret: sails.config.custom.zapierSandboxWebhookSecret,
        }
      })
      .timeout(5000)
      .tolerate(['non200Response', 'requestFailed'], (err)=>{
        // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
        sails.log.warn(`When a user completed the 'What are you using Fleet for' questionnaire step, a lead/contact could not be updated in the CRM for this email address: ${this.req.me.emailAddress}. Raw error: ${err}`);
        return;
      });

      // Set the primary buying situation in the user's session.
      this.req.session.primaryBuyingSituation = primaryBuyingSituation;
    }
    //  ┌─┐┌─┐┌┬┐  ┌─┐┌─┐┬ ┬┌─┐┬ ┬┌─┐┬  ┌─┐┌─┐┬┌─┐┌─┐┬    ┌─┐┌┬┐┌─┐┌─┐┌─┐
    //  └─┐├┤  │   ├─┘└─┐└┬┘│  ├─┤│ ││  │ ││ ┬││  ├─┤│    └─┐ │ ├─┤│ ┬├┤
    //  └─┘└─┘ ┴   ┴  └─┘ ┴ └─┘┴ ┴└─┘┴─┘└─┘└─┘┴└─┘┴ ┴┴─┘  └─┘ ┴ ┴ ┴└─┘└─┘
    // This is how the questionnaire steps/options change a user's psychologicalStage value.
    // 'start': No change
    // 'what-are-you-using-fleet-for': No change
    // 'have-you-ever-used-fleet':
    //  - yes-deployed: » Stage 6
    //  - yes-recently-deployed: » Stage 6
    //  - yes-deployed-local: » Stage 3 (Tried Fleet but don't have a use case)
    //  - yes-deployed-long-time: No change
    //  - no: No change
    // 'how-many-hosts': No change
    // 'will-you-be-self-hosting': No change
    // 'what-are-you-working-on-eo-security'
    //  - no-use-case-yet: » No change
    //  - All other options » Stage 4
    // 'what-does-your-team-manage-eo-it'
    //  - no-use-case-yet: » No change
    //  - All other options » Stage 4
    // 'what-does-your-team-manage-vm'
    //  - no-use-case-yet: » No change
    //  - All other options » Stage 4
    // 'what-do-you-manage-mdm'
    //  - no-use-case-yet: » No change
    //  - All other options » Stage 4
    // 'is-it-any-good': No change
    // 'what-did-you-think'
    //  - deploy-fleet-in-environment » Stage 5
    //  - let-me-think-about-it » No change
    //  - host-fleet-for-me » N/A (currently not selectable, but should set the user's psychologicalStage to stage 5)

    let psychologicalStage = userRecord.psychologicalStage;
    // Get the value of the submitted formData, we do this so we only need to check one variable, instead of (formData.attribute === 'foo');
    let valueFromFormData = _.values(formData)[0];
    if(currentStep === 'have-you-ever-used-fleet') {
      if(['yes-deployed', 'yes-recently-deployed'].includes(valueFromFormData)) {
        // If the user has Fleet deployed, set their stage to 6.
        psychologicalStage = '6 - Has team buy-in';
      } else if(valueFromFormData === 'yes-deployed-local'){
        // If they've tried Fleet locally, set their stage to 3.
        psychologicalStage = '3 - Intrigued';
      }
    } else if(['what-are-you-working-on-eo-security','what-does-your-team-manage-eo-it','what-does-your-team-manage-vm','what-do-you-manage-mdm'].includes(currentStep)){
      if(valueFromFormData === 'no-use-case-yet') {
        // If this user doe not have a use case for Fleet yet, set their psyStage to 3
        psychologicalStage = '3 - Intrigued';
      } else {// Otherwise, they have a use case and will be set to stage 4.
        psychologicalStage = '4 - Has use case';
      }
    } else if(currentStep === 'what-did-you-think') {
      // If the user is ready to deploy Fleet in their work environemnt, then they're ready to get buy-in from their team, so set their psyStage to 5.
      if(valueFromFormData === 'deploy-fleet-in-environment') {
        psychologicalStage = '5 - Personally confident';
      }
      // If the user selects let me think about it, their stage will not change.
    }//ﬁ

    // Send a POST request to Zapier
    await sails.helpers.http.post.with({
      url: 'https://hooks.zapier.com/hooks/catch/3627242/3nltwbg/',
      data: {
        emailAddress: this.req.me.emailAddress,
        firstName: this.req.me.firstName,
        lastName: this.req.me.lastName,
        primaryBuyingSituation: this.req.me.primaryBuyingSituation,
        organization: this.req.me.organization,
        psychologicalStage,
        webhookSecret: sails.config.custom.zapierSandboxWebhookSecret,
      }
    })
    .timeout(5000)
    .tolerate(['non200Response', 'requestFailed'], (err)=>{
      // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
      sails.log.warn(`When a user completed a questionnaire step, a lead/contact could not be updated in the CRM for this email address: ${this.req.me.emailAddress}. Raw error: ${err}`);
      return;
    });
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

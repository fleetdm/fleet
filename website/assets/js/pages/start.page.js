parasails.registerPage('start', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    currentStep: 'start',

    syncing: false,

    // Form data
    formData: {
      'start': {stepCompleted: true},
      'what-are-you-using-fleet-for': {},
      'have-you-ever-used-fleet': {},
      'how-many-hosts': {},
      'will-you-be-self-hosting': {},
      'what-are-you-working-on-eo-security': {},
      'is-it-any-good': {stepCompleted: true},
      'what-did-you-think': {},
    },
    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    formRules: {},
    primaryBuyingSituationFormRules: {
      primaryBuyingSituation: {required: true}
    },
    isUsingFleetFormRules: {
      fleetUseStatus: {required: true}
    },
    numberOfHostsFormRules: {
      numberOfHosts: {required: true}
    },
    hostingFleetFormRules: {
      willSelfHost: {required: true}
    },
    endpointOpsSecurityWorkingOnFormRules: {
      endpointOpsSecurityUseCase: {required: true}
    },
    endpointOpsSecurityIsItAnyGoodFormRules: {
      isItAnyGood: {required: true}
    },
    endpointOpsSecurityWhatDidYouThinkFormRules: {
      whatDidYouThink: {required: true}
    },
    previouslyAnsweredQuestions: {},

    // Server error state for the forms
    cloudError: '',

    // Success state when form has been submitted
    cloudSuccess: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
    if(this.currentStep !== 'start'){
      this.prefillPreviousAnswers();
    }
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    handleSubmittingForm: async function(argins) {
      let formDataForThisStep = _.clone(argins);
      let nextStep = this.getNextStep();
      let getStartedProgress = await Cloud.saveQuestionnaireProgress.with({
        currentStep: this.currentStep,
        formData: formDataForThisStep,
      });
      this.previouslyAnsweredQuestions[this.currentStep] = getStartedProgress[this.currentStep];
      this.syncing = false;
      this.currentStep = nextStep;
    },
    clickGoToPreviousStep: async function() {
      switch(this.currentStep) {
        case 'have-you-ever-used-fleet':
          this.currentStep = 'what-are-you-using-fleet-for';
          break;
        case 'how-many-hosts':
          this.currentStep = 'have-you-ever-used-fleet';
          break;
        case 'will-you-be-self-hosting':
          this.currentStep = 'how-many-hosts';
          break;
        case 'self-hosted-deploy':
          this.currentStep = 'will-you-be-self-hosting';
          break;
        case 'managed-cloud-for-growing-deployments':
          this.currentStep = 'will-you-be-self-hosting';
          break;
        case 'what-are-you-working-on-eo-security':
          this.currentStep = 'have-you-ever-used-fleet';
          break;
        case 'is-it-any-good':
          this.currentStep = 'what-are-you-working-on-eo-security';
          break;
        case 'lets-talk-to-your-team':
          this.currentStep = 'how-many-hosts';
          break;
        case 'welcome-to-fleet':
          this.currentStep = 'have-you-ever-used-fleet';
          break;
        case 'deploy-fleet-in-your-environment':
          this.currentStep = 'what-did-you-think';
          break;
        case 'what-did-you-think':
          this.currentStep = 'is-it-any-good';
          break;
      }
    },
    getNextStep: function() {
      let nextStepInForm;
      switch(this.currentStep) {
        case 'start':
          nextStepInForm = 'what-are-you-using-fleet-for';
          break;
        case 'what-are-you-using-fleet-for':
          nextStepInForm = 'have-you-ever-used-fleet';
          break;
        case 'have-you-ever-used-fleet':
          let fleetUseStatus = this.formData['have-you-ever-used-fleet'].fleetUseStatus;
          let primaryBuyingSituation = this.formData['what-are-you-using-fleet-for'].primaryBuyingSituation;
          if(fleetUseStatus === 'yes-recently-deployed' || fleetUseStatus === 'yes-deployed') {
            nextStepInForm = 'how-many-hosts';
          } else {
            if(primaryBuyingSituation === 'eo-security'){
              nextStepInForm = 'what-are-you-working-on-eo-security';
            } else {
              nextStepInForm = 'welcome-to-fleet';
            }
          }
          break;
        case 'how-many-hosts':
          if(this.formData['how-many-hosts'].numberOfHosts === '1-100' ||
            this.formData['how-many-hosts'].numberOfHosts === '100-700') {
            nextStepInForm = 'will-you-be-self-hosting';
          } else {
            nextStepInForm = 'lets-talk-to-your-team';
          }
          break;
        case 'will-you-be-self-hosting':
          if(this.formData['will-you-be-self-hosting'].willSelfHost === 'true'){
            nextStepInForm = 'self-hosted-deploy';
          } else {
            nextStepInForm = 'managed-cloud-for-growing-deployments';
          }
          break;
        case 'what-are-you-working-on-eo-security':
          nextStepInForm = 'is-it-any-good';
          break;
        case 'is-it-any-good':
          nextStepInForm = 'what-did-you-think';
          break;
        case 'what-did-you-think':
          if(this.formData['what-did-you-think'].whatDidYouThink === 'let-me-think-about-it'){
            nextStepInForm = 'is-it-any-good';
          } else {
            nextStepInForm = 'deploy-fleet-in-your-environment';
          }
          break;
      }
      return nextStepInForm;
    },
    clickGoToCalendly: function() {
      window.location = `https://calendly.com/fleetdm/talk-to-us?email=${encodeURIComponent(this.me.emailAddress)}&name=${encodeURIComponent(this.me.firstName+' '+this.me.lastName)}`;
    },
    clickGoToContactPage: function() {
      window.location = `/contact?prefillFormDataFromUserRecord`;
    },
    clickClearOneFormError: function(field) {
      if(this.formErrors[field]){
        this.formErrors = _.omit(this.formErrors, field);
      }
    },
    prefillPreviousAnswers: function() {
      if(!_.isEmpty(this.previouslyAnsweredQuestions)){
        for(let step in this.previouslyAnsweredQuestions){
          this.formData[step] = this.previouslyAnsweredQuestions[step];
        }
        this.currentStep = this.getNextStep();
      }
    },
  }
});

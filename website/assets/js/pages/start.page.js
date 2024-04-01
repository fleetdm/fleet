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
      'what-are-you-working-on': {
        endpointOpsSecurityUseCase: undefined,
      },
      'what-are-you-working-on-eo-security': {},
      'is-it-any-good': {stepCompleted: true},
      'what-did-you-think': {},
    },



    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    formLessStepRules: {},
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

    endpointOpsSecurityWorkinOnFormRules: {
      endpointOpsSecurityUseCase: {required: true}
    },

    endpointOpsSecurityIsItAnyGoodFormRules: {
      isItAnyGood: {required: true}
    },

    endpointOpsSecurityWhatDidYouThinkFormRules: {
      whatDidYouThink: {required: true}
    },
    previouslyAnsweredQuestions: {},
    // Server error state for the form
    cloudError: '',

    // Success state when form has been submitted
    cloudSuccess: false,
    prefilledAnswers: {},
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
      console.log('argins:',argins);
      let formDataForThisStep = _.clone(argins);
      let responseFromEndpoint = await Cloud.saveQuestionnaireProgress.with({
        currentStep: this.currentStep,
        formData: formDataForThisStep,
      });
      console.log(responseFromEndpoint);
      this.previouslyAnsweredQuestions[this.currentStep] = responseFromEndpoint.previouslyAnsweredQuestions[this.currentStep];
      this.syncing = false;
      this.currentStep = responseFromEndpoint.currentStep;
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
          this.currentStep = 'will-you-be-self-hosting';
          break;
      }
    },
    clickGoToCalendly: function() {
      window.location = `https://calendly.com/fleetdm/talk-to-us?email=${encodeURIComponent(this.me.emailAddress)}&name=${encodeURIComponent(this.me.firstName+' '+this.me.lastName)}`;
    },
    prefillPreviousAnswers: async function() {
      for(let formKey in this.previouslyAnsweredQuestions){
        this.formData[formKey] = this.previouslyAnsweredQuestions[formKey];
      }
    },
  }
});

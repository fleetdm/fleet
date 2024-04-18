parasails.registerPage('contact', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    formToDisplay: 'talk-to-us',
    primaryBuyingSituation: undefined,
    // Main syncing/loading state for this page.
    syncing: false,

    // Form data
    formData: { /* … */ },

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    // "talk to us" Form rules
    talkToUsFormRules: {
      emailAddress: {isEmail: true, required: true},
      firstName: {required: true},
      lastName: {required: true},
      organization: {required: true},
      primaryBuyingSituation: {required: true},
      numberOfHosts: {required: true},
    },
    // Contact form rules
    contactFormRules: {
      emailAddress: {isEmail: true, required: true},
      firstName: {required: true},
      lastName: {required: true},
      message: {required: false},
    },
    formDataToPrefillForLoggedInUsers: {},

    // Server error state for the form
    cloudError: '',

    // Success state when form has been submitted
    cloudSuccess: false,

    // For personalizing the message at the top of the contact form.
    buyingStage: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(this.formToShow === 'contact'){
      this.formToDisplay = this.formToShow;
    }
    if(this.primaryBuyingSituation){ // If the user has a priamry buying situation set in their sesssion, pre-fill the form.
      // Note: this will be overriden if the user is logged in and has a primaryBuyingSituation set in the database.
      this.formData.primaryBuyingSituation = this.primaryBuyingSituation;
    }
    if(this.me){// prefill from database
      this.formDataToPrefillForLoggedInUsers.emailAddress = this.me.emailAddress;
      this.formDataToPrefillForLoggedInUsers.firstName = this.me.firstName;
      this.formDataToPrefillForLoggedInUsers.lastName = this.me.lastName;
      this.formDataToPrefillForLoggedInUsers.organization = this.me.organization;
      // Only prefil this information if the user has this value set.
      if(this.me.primaryBuyingSituation) {
        this.formDataToPrefillForLoggedInUsers.primaryBuyingSituation = this.me.primaryBuyingSituation;
      }
      this.formData = _.clone(this.formDataToPrefillForLoggedInUsers);
      // If this user has submitted the /start questionnaire, determine their buying stage based on the answers they provided
      if(!_.isEmpty(this.me.getStartedQuestionnaireAnswers)) {
        let getStartedQuestionnaireAnswers = _.clone(this.me.getStartedQuestionnaireAnswers);
        if(getStartedQuestionnaireAnswers['have-you-ever-used-fleet']) {
          // If the user has Fleet deployed, then we'll assume they're stage five.
          if(getStartedQuestionnaireAnswers['have-you-ever-used-fleet'].fleetUseStatus === 'yes-deployed' ||
            getStartedQuestionnaireAnswers['have-you-ever-used-fleet'].fleetUseStatus === 'yes-recently-deployed') {
            this.buyingStage = 'five';
          }
        }
        if(getStartedQuestionnaireAnswers['what-did-you-think']){
          // If this user has completed the "What did you think" step and wants to self-host Fleet, we'll assume theyre stage four.
          if(getStartedQuestionnaireAnswers['what-did-you-think'].whatDidYouThink === 'deploy-fleet-in-environment'){
            this.buyingStage = 'four';
          }
        }
      }
    }
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    submittedContactForm: async function() {

      // Show the success message.
      this.cloudSuccess = true;

    },
    submittedTalkToUsForm: async function() {
      this.syncing = true;
      if(this.formData.numberOfHosts > 700){
        window.location = `https://calendly.com/fleetdm/talk-to-us?email=${encodeURIComponent(this.formData.emailAddress)}&name=${encodeURIComponent(this.formData.firstName+' '+this.formData.lastName)}`;
      } else {
        window.location = `https://calendly.com/fleetdm/chat?email=${encodeURIComponent(this.formData.emailAddress)}&name=${encodeURIComponent(this.formData.firstName+' '+this.formData.lastName)}`;
      }
    },

    clickSwitchForms: function(form) {
      if(this.prefillFormDataFromUserRecord){
        this.formData = _.clone(this.formDataToPrefillForLoggedInUsers);
      } else {
        this.formData = {};
      }
      this.formErrors = {};
      this.cloudError = '';
      this.formToDisplay = form;
    }

  }
});

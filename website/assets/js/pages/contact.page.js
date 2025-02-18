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

    // For personalizing the message at the top of the contact form for logged-in users.
    psychologicalStage: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(this.formToShow === 'contact'){
      this.formToDisplay = this.formToShow;
    } else if(!this.primaryBuyingSituation){
      // Otherwise, default to the formToShow value from the page's controller.
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
      // Only prefil this information if the user has this value set to a value that is not VM.
      if(this.me.primaryBuyingSituation && this.me.primaryBuyingSituation !== 'vm') {
        this.formDataToPrefillForLoggedInUsers.primaryBuyingSituation = this.me.primaryBuyingSituation;
      }
      this.formData = _.clone(this.formDataToPrefillForLoggedInUsers);
      this.psychologicalStage = this.me.psychologicalStage;
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
      if(typeof gtag !== 'undefined'){
        gtag('event','fleet_website__contact_forms');
      }
      if(typeof window.lintrk !== 'undefined') {
        window.lintrk('track', { conversion_id: 18587089 });// eslint-disable-line camelcase
      }
      if(typeof analytics !== 'undefined'){
        analytics.track('fleet_website__contact_forms');
      }
      // Show the success message.
      this.cloudSuccess = true;

    },
    submittedTalkToUsForm: async function() {
      this.syncing = true;
      if(typeof gtag !== 'undefined'){
        gtag('event','fleet_website__contact_forms');
      }
      if(typeof window.lintrk !== 'undefined') {
        window.lintrk('track', { conversion_id: 18587089 });// eslint-disable-line camelcase
      }
      if(typeof analytics !== 'undefined'){
        analytics.track('fleet_website__contact_forms');
      }
      if(this.formData.numberOfHosts > 300){
        this.goto(`https://calendly.com/fleetdm/talk-to-us?email=${encodeURIComponent(this.formData.emailAddress)}&name=${encodeURIComponent(this.formData.firstName+' '+this.formData.lastName)}`);
      } else {
        this.goto(`https://calendly.com/fleetdm/chat?email=${encodeURIComponent(this.formData.emailAddress)}&name=${encodeURIComponent(this.formData.firstName+' '+this.formData.lastName)}`);
      }
    },

    clickSwitchForms: function(form) {
      if(this.me){
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

parasails.registerPage('contact', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    formToDisplay: 'talk-to-us',
    audience: undefined,
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

    // Server error state for the form
    cloudError: '',

    // Success state when form has been submitted
    cloudSuccess: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(this.formToShow === 'contact'){
      this.formToDisplay = this.formToShow;
    }
    if(window.location.search){
      window.history.replaceState({}, document.title, '/contact' );
    }
    if(this.primaryBuyingSituation){
      this.audience = this.primaryBuyingSituation;
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
        window.location = `https://calendly.com/fleetdm/talk-to-us?email=${encodeURIComponent(this.formData.emailAddress)}&name=${this.formData.firstName}+${this.formData.lastName}`;
      } else {
        window.location = `https://calendly.com/fleetdm/chat?email=${encodeURIComponent(this.formData.emailAddress)}&name=${this.formData.firstName}+${this.formData.lastName}`;
      }
    },

    clickSwitchForms: function(form) {
      this.formData = {};
      this.formErrors = {};
      this.cloudError = '';
      this.formToDisplay = form;
    }

  }
});

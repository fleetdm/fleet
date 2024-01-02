parasails.registerPage('launch-party', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    formData: { /* … */ },

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    // Form rules
    formRules: {
      firstName: {required: true },
      lastName: {required: true },
      emailAddress: {required: true, isEmail: true},
    },
    cloudError: '',
    // Syncing / loading state
    syncing: false,
    showSignupFormSuccess: false,
    // Modal

    modal: '',
    showAlternateWaitlistText: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

    //…
  },
  mounted: async function() {

    if(this.showForm) {
      this.modal = 'happy-hour-waitlist';
      if(!_.isEmpty(this.formDataToPrefill)){
        // If the user came here via a personalized link in an email, we'll prefill the form with the user information (if provided)
        this.formData = this.formDataToPrefill;
        this.showAlternateWaitlistText = true;
      }
    }

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickOpenModal: function() {
      this.modal = 'happy-hour-waitlist';
    },
    closeModal: async function () {
      this.modal = '';
      await this._resetForms();
    },
    typeClearOneFormError: async function(field) {
      if(this.formErrors[field]){
        this.formErrors = _.omit(this.formErrors, field);
      }
    },
    submittedForm: function() {
      this.showSignupFormSuccess = true;
    },
    _resetForms: async function() {
      this.cloudError = '';
      this.formData = {};
      this.formErrors = {};
      await this.forceRender();
    },
  }
});

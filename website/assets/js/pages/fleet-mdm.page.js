parasails.registerPage('device-management', {
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
      fullName: {required: true },
      emailAddress: {required: true, isEmail: true},
      jobTitle: {required: true },
      numberOfHosts: {required: true },
    },
    cloudError: '',
    // Syncing / loading state
    syncing: false,
    showSignupFormSuccess: false,
    // Modal

    modal: '',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickOpenSignupModal: function() {
      this.modal = 'beta-signup';
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
      this.formRules = {};
      this.formErrors = {};
      await this.forceRender();
    },
  }
});

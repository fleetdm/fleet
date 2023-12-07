parasails.registerPage('register', {
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
      firstName: {required: true},
      lastName: {required: true},
      organization: {required: true},
      emailAddress: {required: true, isEmail: true},
      password: {required: true, minLength: 8},
    },
    // Syncing / loading state
    syncing: false,
    // Server error state
    cloudError: '',
    // Modal
    modal: '',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // If the user navigated to this page from the 'try it now' button, we'll strip the '?tryitnow' from the url.
    if(window.location.search){
      // https://caniuse.com/mdn-api_history_replacestate
      window.history.replaceState({}, document.title, '/try-fleet/register' );
    }
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    // Using handle-submitting to add firstName, and lastName values to our formData before sending it to signup.js
    handleSubmittingRegisterForm: async function(argins) {
      argins.signupReason = 'Try Fleet';
      return await Cloud.signup.with(argins);
    },

    // After the form is submitted, we'll redirect the user to the fleetctl preview page.
    submittedRegisterForm: async function() {
      this.syncing = true;
      window.location = '/try-fleet/fleetctl-preview';
    },

    clickOpenVideoModal: function() {
      this.modal = 'video';
    },

    closeModal: function() {
      this.modal = '';
    },
  }
});

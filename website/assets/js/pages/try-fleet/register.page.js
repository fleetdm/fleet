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
      // Creating a copy of the formdata to submit to the signup action. Otherwise, any changes to the formData before we call our signup action would be visible to the user.
      let signupArgins = _.clone(argins);
      if(!this.formData.firstName){
        if(this.formData.lastName) {// If a user provided a lastName but no firstName, we'll set the firstName to '?' instead of a fragment of the users email address.
          signupArgins.firstName = '?'
        } else {
          signupArgins.firstName = argins.emailAddress.split('@')[0];
        }
      }
      if(!this.formData.lastName) {
        if(this.formData.firstName) {// If a user provided a firstName but no lastName, we'll set the lastName to '?' instead of a fragment of the users email address.
          signupArgins.lastName = '?';
        }
        signupArgins.lastName = argins.emailAddress.split('@')[1];
      }
      signupArgins.signupReason = 'Try Fleet Sandbox';
      return await Cloud.signup.with(signupArgins);
    },

    // After the form is submitted, we'll redirect the user to their Fleet sandbox instance.
    submittedRegisterForm: async function() {
      this.syncing = true;
      window.location = '/try-fleet/sandbox';
    },

    clickOpenVideoModal: function() {
      this.modal = 'video';
    },

    closeModal: function() {
      this.modal = '';
    },
  }
});

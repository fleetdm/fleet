parasails.registerPage('signup', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Form data
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
    // For displaying the full signup form.
    showFullForm: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // Removing the query string for users redirected to this page by the /try-fleet/explore-data pages.
    // FUTURE: remove this when that view-query-report is updated.
    if(window.location.search){
      window.history.replaceState({}, document.title, '/register' );
    }
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickResetForm: async function() {
      this.cloudError = '';
      this.formErrors = {};
      this.showFullForm = true;
      await this.forceRender();
    },

    typeClearOneFormError: async function(field) {
      this.showFullForm = true;
      if(this.formErrors[field]){
        this.formErrors = _.omit(this.formErrors, field);
      }
    },

    submittedSignUpForm: async function() {
      // redirect to the /start page.
      // > (Note that we re-enable the syncing state here.  This is on purpose--
      // > to make sure the spinner stays there until the page navigation finishes.)
      this.syncing = true;
      window.location = '/start';
    }


  }
});

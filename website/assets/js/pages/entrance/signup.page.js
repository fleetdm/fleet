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
    // For redirecting users coming from the "Get your license" link to the license dispenser.
    loginSlug: '/login',
    pageToRedirectToAfterRegistration: '/start#signup',
    primaryBuyingSituation: undefined
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // If we're redirecting this user to the license dispenser after they sign up, modify the link to the login page and the pageToRedirectToAfterRegistration
    if(window.location.hash && window.location.hash === '#purchaseLicense'){
      this.loginSlug = '/login#purchaseLicense';
      this.pageToRedirectToAfterRegistration = '/new-license#signup';
      window.location.hash = '';
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

    submittedSignUpForm: async function() {// When the server says everything worked…
      // Track a "key event" in Google Analytics. (?  but don't we do that when we call analytics.track() [segment] later on in start.page.js?  TODO: eric help please – I suspect this one is either duplicate OR it's actually writing to Google Ads, and not to Google Analytics.  I'm pretty sure segment's .track() is what writes to google analytics.)
      // > Naming convention:  (like sails config)
      // > "Website - Sign up" becomes "fleet_website__sign_up"  (double-underscore representing hierarchy)
      if(window.gtag !== undefined){
        window.gtag('event','fleet_website__sign_up');
      }

      // Track a "conversion" in LinkedIn Campaign Manager.
      if(window.lintrk !== undefined) {
        window.lintrk('track', { conversion_id: 18587097 });// eslint-disable-line camelcase
      }

      // Redirect to the /start page.
      // > (Note that we re-enable the syncing state here.  This is on purpose--
      // > to make sure the spinner stays there until the page navigation finishes.)
      this.syncing = true;
      this.goto(this.pageToRedirectToAfterRegistration);// « / start if the user came here from the start now button, or customers/new-license if the user came here from the "Get your license" link.
    }


  }
});

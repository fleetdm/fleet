parasails.registerPage('login', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {


    formToDisplay: 'signup',
    // Shared by forms
    syncing: false,
    cloudError: undefined,
    // Signup form
    signupFormData: {},
    signupFormErrors: {},
    signupFormRules: {
      firstName: {required: true},
      lastName: {required: true},
      emailAddress: {required: true, isEmail: true},
      password: {
        required: true,
        minLength: 12,
        maxLength: 48,
        custom: (value)=>{
          return value.match(/^(?=.*\d)(?=.*[^A-Za-z0-9]).{12,48}$/);
        }
      },
    },
    // Login form
    loginFormData: {},
    loginFormErrors: {},
    loginFormRules: {
      emailAddress: {required: true, isEmail: true},
      password: {required: true},
    },


    // Main syncing/loading state for this page.
    // syncing: false,

    // Form data
    formData: {
      rememberMe: true,
    },

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    // A set of validation rules for our form.
    // > The form will not be submitted if these are invalid.
    formRules: {
      emailAddress: { required: true, isEmail: true },
      password: { required: true },
    },

    // Server error state for the form
    // cloudError: '',
    showCustomerLogin: true,
    // For redirecting users coming from the "Get your license" link to the license dispenser.
    registerSlug: '/register',
    pageToRedirectToAfterLogin: '/try',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(window.location.search === '?admin') {
      this.showCustomerLogin = false;
    }

    if(window.location.hash){
      // If we're redirecting this user to the license dispenser after they log in, modify the link to the /register page and the pageToRedirectToAfterLogin.
      if(window.location.hash === '#purchaseLicense'){
        this.registerSlug = '/register#purchaseLicense';
        this.pageToRedirectToAfterLogin = '/new-license';
        window.location.hash = '';
      // If we're redirecting this user to the contact form after they log in, modify the link to the /register page and the pageToRedirectToAfterLogin.
      } else if(window.location.hash === '#contact'){
        this.registerSlug = '/register';
        this.pageToRedirectToAfterLogin = '/contact?sendMessage';
        window.location.hash = '';
      } else if(window.location.hash === '#tryfleet'){
        this.registerSlug = '/register#tryfleet';
        this.pageToRedirectToAfterLogin = '/try-fleet';
        window.location.hash = '';
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
    switchForm(form) {
      this.formToDisplay = form;
    },

    clickResetForm: async function() {
      this.cloudError = '';
      this.signupFormErrors = {};
      await this.forceRender();
    },

    typeClearOneFormError: async function(field) {
      if(this.signupFormErrors[field]){
        this.signupFormErrors = _.omit(this.signupFormErrors, field);
      }
    },
    submittedSignupForm: async function(){
      this.syncing = true;
      // Track a "key event" in Google Analytics.
      // > Naming convention:  (like sails config)
      // > "Website - Sign up" becomes "fleet_website__sign_up"  (double-underscore representing hierarchy)
      if(window.gtag !== undefined){
        window.gtag('event','fleet_website__sign_up');
      }

      // Track a "conversion" in LinkedIn Campaign Manager.
      if(window.lintrk !== undefined) {
        window.lintrk('track', { conversion_id: 18587097 });// eslint-disable-line camelcase
      }
      this.goto('/try');
    },
    submittedLoginForm: async function() {
      this.syncing = true;
      this.goto('/try');
    },
    submittedForm: async function() {
      // Redirect to the /start page on success.
      // > (Note that we re-enable the syncing state here.  This is on purpose--
      // > to make sure the spinner stays there until the page navigation finishes.)
      this.syncing = true;
      this.goto(this.pageToRedirectToAfterLogin);
    },

    clickGoBack: function () {
      if(window.navigation && window.navigation.canGoBack){
        window.navigation.back();
      } else {
        this.goto('/');
      }
    }

  }
});

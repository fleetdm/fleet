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
      primaryBuyingSituation: {required: true},
    },
    // Syncing / loading state
    syncing: false,
    // Server error state
    cloudError: '',
    // For displaying the full signup form.
    showFullForm: false,
    exploreDataRedirectSlug: undefined,
    // Used for the 'I have an account' link
    loginSlug: '/login',
    // Possible /try-fleet/explore-data/ redirects
    redirectSlugsByTargetPlatform: {
      'macos': 'macos/account_policy_data',
      'windows': 'windows/appcompat_shims',
      'linux': 'linux/apparmor_events',
    },
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // If the user navigated to this page from an /explore-data page, we'll keep track of the page this user came from so we can redirect them, and we'll strip all query parameters from the URL.
    if(window.location.search){
      // https://caniuse.com/mdn-api_urlsearchparams_get
      let possibleSearchParamsToFilterBy = new URLSearchParams(window.location.search);
      let posibleRedirect = possibleSearchParamsToFilterBy.get('targetPlatform');
      // If the provided platform matches a key in the userFriendlyPlatformNames array, we'll set this.selectedPlatform.
      if(posibleRedirect && this.redirectSlugsByTargetPlatform[posibleRedirect] !== undefined){
        this.loginSlug +=`?targetPlatform=${posibleRedirect}`;
        this.exploreDataRedirectSlug = `/try-fleet/explore-data/${this.redirectSlugsByTargetPlatform[posibleRedirect]}`;
      }
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
      if(this.exploreDataRedirectSlug){
        window.location = this.exploreDataRedirectSlug;
      } else {
        window.location = '/start';
      }
    }


  }
});

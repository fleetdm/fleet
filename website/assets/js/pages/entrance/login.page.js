parasails.registerPage('login', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Main syncing/loading state for this page.
    syncing: false,

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
    cloudError: '',
    showCustomerLogin: true,

    // For redirecting users who come to this page from a /try-fleet/explore-data/* page back to the page they were visiting before they were redirected.
    exploreDataRedirectSlug: undefined,
    // Used for the 'create an account' link
    registrationSlug: '/register',
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
    if(window.location.search === '?admin') {
      this.showCustomerLogin = false;
    }
    // If the user navigated to this page from an /explore-data page, we'll keep track of the page this user came from so we can redirect them, and we'll strip all query parameters from the URL.
    if(window.location.search) {
      // https://caniuse.com/mdn-api_urlsearchparams_get
      let possibleSearchParamsToFilterBy = new URLSearchParams(window.location.search);
      let posibleRedirect = possibleSearchParamsToFilterBy.get('targetPlatform');
      // If the provided platform matches a key in the userFriendlyPlatformNames array, we'll set this.selectedPlatform.
      if(posibleRedirect && this.redirectSlugsByTargetPlatform[posibleRedirect] !== undefined){
        this.registrationSlug +=`?targetPlatform=${posibleRedirect}`;
        this.exploreDataRedirectSlug = `/try-fleet/explore-data/${this.redirectSlugsByTargetPlatform[posibleRedirect]}`;
      }
      window.history.replaceState({}, document.title, '/login' );
    }
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    submittedForm: async function() {
      // Redirect to the logged-in dashboard on success.
      // > (Note that we re-enable the syncing state here.  This is on purpose--
      // > to make sure the spinner stays there until the page navigation finishes.)
      this.syncing = true;
      if(this.exploreDataRedirectSlug){
        window.location = this.exploreDataRedirectSlug;
      } else {
        window.location = '/start';
      }

    },

  }
});

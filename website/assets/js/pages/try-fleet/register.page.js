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
    // For redirecting users who come to this page from a /try-fleet/explore-data/* page back to the page they were visiting before they were redirected.
    exploreDataRedirectSlug: undefined,
    // Used for the 'I have an account' link
    loginSlug: '/try-fleet/login',
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
      if(this.exploreDataRedirectSlug){
        window.location = this.exploreDataRedirectSlug;
      } else {
        window.location = '/try-fleet/explore-data';
      }
    },

    clickOpenVideoModal: function() {
      this.modal = 'video';
    },

    closeModal: function() {
      this.modal = '';
    },
  }
});

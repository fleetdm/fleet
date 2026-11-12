parasails.registerPage('dashboard', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Main syncing/loading state for this page.
    syncing: false,

    // Form data
    formData: {},

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    // Form rules
    formRules: {},

    // Server error state for the form
    cloudError: '',
    modal: '',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickEditButton: function() {
      this.formData = {
        firstName: this.me.firstName,
        lastName: this.me.lastName,
        organization: this.me.organization,
        emailAddress: this.me.emailAddress,
      };
      this.formRules = {
        firstName: {required: true},
        lastName: {required: true},
        organization: {required: true},
        emailAddress: {required: true, isEmail: true},
      };
      this.modal = 'update-profile';
    },

    clickChangePassword: function() {
      this.formData = {};
      this.formRules = {
        oldPassword: {required: true},
        newPassword: {
          required: true,
          minLength: 12,
          maxLength: 48,
          custom: (value)=>{
            return value.match(/^(?=.*\d)(?=.*[^A-Za-z0-9]).{12,48}$/);
          }
        },
      };
      this.modal = 'update-password';
    },

    clickCopyLicenseKey: function() {
      $('[purpose="copied-notification"]').finish();
      $('[purpose="copied-notification"]').fadeIn(100).delay(2000).fadeOut(500);
      // https://caniuse.com/mdn-api_clipboard_writetext
      navigator.clipboard.writeText(this.thisSubscription.fleetLicenseKey);
    },

    clickExpandLicenseKey: function() {
      $('[purpose="license-key"]').toggleClass('show-overflow');
    },

    closeModal: async function() {
      // Dismiss modal
      this.modal = '';
      await this._resetForms();
    },

    submittedUpdateProfileForm: async function() {
      this.syncing = true;
      Object.assign(this.me, _.pick(this.formData, ['firstName', 'lastName', 'organization', 'emailAddress']));
      this.modal = '';
      await this._resetForms();
      this.syncing = false;
    },

    submittedUpdatePasswordForm: async function() {
      this.syncing = true;
      this.modal = '';
      await this._resetForms();
      this.syncing = false;
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

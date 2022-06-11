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

    clickUpdateBillingCardButton: function() {
      this.modal = 'update-billing-card';
      this.formData = { newPaymentSource: undefined };
      this.formRules = { newPaymentSource: {required: true}};
    },

    clickChangePassword: function() {
      this.formData = {};
      this.formRules = {
        oldPassword: {required: true},
        newPassword: {required: true, minLength: 8},
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

    // clickRemoveCardButton: async function() {
    //   this.modal = 'remove-billing-card';
    //   this.formData.stripeToken = '';
    // },

    closeModal: async function() {
      // Dismiss modal
      this.modal = '';
      await this._resetForms();
    },

    handleSubmittingUpdateBillingCard: async function(argins) {
      var newPaymentSource = argins.newPaymentSource;
      await Cloud.updateBillingCard.with(newPaymentSource);
    },

    submittedUpdateBillingCard: async function() {
      Object.assign(this.me, _.pick(this.formData.newPaymentSource, ['billingCardLast4', 'billingCardBrand', 'billingCardExpMonth', 'billingCardExpYear']));
      this.me.hasBillingCard = true;
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

    // submittedRemoveCardForm: async function() {
    //   // Update billing info on success.
    //   this.me.billingCardLast4 = undefined;
    //   this.me.billingCardBrand = undefined;
    //   this.me.billingCardExpMonth = undefined;
    //   this.me.billingCardExpYear = undefined;
    //   this.me.hasBillingCard = false;

    //   // Close the modal and clear it out.
    //   this.closeModal();
    // },

    _resetForms: async function() {
      this.cloudError = '';
      this.formData = {};
      this.formRules = {};
      this.formErrors = {};
      await this.forceRender();
    },
  }
});

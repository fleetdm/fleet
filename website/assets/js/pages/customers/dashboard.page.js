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
    // defining the license key here until I get live keys going.
    licenseKey: '1234 1234 1234 1234 1234'
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
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
        emailAddress: {required: true, isEmail: true},
      };
      this.modal = 'update-profile';
    },

    closeModal: async function() {
      // Dismiss modal
      this.modal = '';
      await this._resetForms();
    },
    _resetForms: async function() {
      this.cloudError = '';
      this.formData = {};
      this.formRules = {};
      this.formErrors = {};
      await this.forceRender();
    },

    submittedRemoveCardForm: async function() {

      // Update billing info on success.
      this.me.billingCardLast4 = undefined;
      this.me.billingCardBrand = undefined;
      this.me.billingCardExpMonth = undefined;
      this.me.billingCardExpYear = undefined;
      this.me.hasBillingCard = false;

      // Close the modal and clear it out.
      this.closeModal();
    },

    clickCopyLicenseKey: function() {
      navigator.clipboard.writeText(this.licenseKey);
    },

    clickRemoveCardButton: async function() {
      this.modal = 'remove-billing-card';
      this.formData.stripeToken = '';
    },

    clickUpdateBillingCardButton: function() {
      this.modal = 'update-billing-card';
      this.formData = { newPaymentSource: undefined };
      this.formRules = { newPaymentSource: {required: true}};
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
      // Redirect to the account page on success.
      // > (Note that we re-enable the syncing state here.  This is on purpose--
      // > to make sure the spinner stays there until the page navigation finishes.)
      this.syncing = true;
      window.location = '/customers/dashboard';
    },
  }
});

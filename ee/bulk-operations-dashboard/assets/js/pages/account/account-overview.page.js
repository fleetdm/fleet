parasails.registerPage('account-overview', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    isBillingEnabled: false,

    hasBillingCard: false,

    // Syncing/loading states for this page.
    syncingOpenCheckout: false,
    syncingUpdateCard: false,
    syncingRemoveCard: false,

    // For <ajax-form>
    formData: { /* … */ },
    formRules: { /* … */ },
    formErrors: { /* … */ },
    cloudError: '',
    syncing: '',

    // For <modal>:
    modal: '',

  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function (){
    _.extend(this, window.SAILS_LOCALS);

    this.isBillingEnabled = !!this.stripePublishableKey;

    // Determine whether there is billing info for this user.
    this.me.hasBillingCard = (
      this.me.billingCardBrand &&
      this.me.billingCardLast4 &&
      this.me.billingCardExpMonth &&
      this.me.billingCardExpYear
    );
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickUpdateBillingCardButton: function() {
      this.modal = 'update-billing-card';
      this.formData = { newPaymentSource: undefined };
      this.formRules = { newPaymentSource: {required: true}};
    },

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

    _resetForms: async function() {
      this.cloudError = '';
      this.formData = {};
      this.formRules = {};
      this.formErrors = {};
      await this.forceRender();
    },

    clickRemoveCardButton: async function() {
      this.modal = 'remove-billing-card';
      this.formData.stripeToken = '';
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

  }
});

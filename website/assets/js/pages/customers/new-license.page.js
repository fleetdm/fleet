parasails.registerPage('new-license', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Form data
    formData: { /* … */ },

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    quoteFormRules: {
      numberOfHosts: {required: true},
    },

    billingFormRules: {
      paymentSource: {required: true},
      selfHostedAcknowledgment: {required: true, is: true},
    },

    // Syncing / loading state
    syncing: false,

    // Server error state
    cloudError: '',


    quotedPrice: undefined,
    numberOfHostsQuoted: undefined,
    showBillingForm: false,
    showQuotedPrice: false,
    showAdditionalBillingFormInputs: false,
    // Success state when the billing form has been submitted
    showSuccessMessage: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {

    // If this user's signupReason is 'Try Fleet Sandbox' we'll need some additional information to complete this order.
    if(this.me.signupReason === 'Try Fleet Sandbox') {
      this.showAdditionalBillingFormInputs = true;
      this.billingFormRules.organization = {required: true};
      this.billingFormRules.firstName = {required: true};
      this.billingFormRules.lastName = {required: true};
    }

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    submittedPaymentForm: async function() {
      // After payment is submitted, Display a success message and let them navigate to the dashboard
      this.showSuccessMessage = true;
      await this.forceRender();
      this.$focus('[purpose="submit-button"]');
    },

    clickGoToDashboard: async function() {
      this.syncing = true;
      this.goto('/customers/dashboard?order-complete');
    },

    submittedQuoteForm: async function(quote) {
      this.showQuotedPrice = true;
      this.quotedPrice = quote.quotedPrice;
      this.numberOfHostsQuoted = quote.numberOfHosts;
      if(quote.numberOfHosts < 700) {
        this.formData.quoteId = quote.id;
        this.showBillingForm = true;
      }
      await this.forceRender();
    },

    clickClearOneFormError: async function(field) {
      if(this.formErrors[field]){
        this.formErrors = _.omit(this.formErrors, field);
      }
    },

    clickScheduleDemo: async function() {
      this.syncing = true;
      // Note: we keep loading spinner present indefinitely so that it is apparent that a new page is loading
      this.goto(`https://calendly.com/fleetdm/talk-to-us?email=${encodeURIComponent(this.me.emailAddress)}&name=${encodeURIComponent(this.me.firstName+' '+this.me.lastName)}`);
    },

    clickResetForm: async function() {
      // When the "X" is clicked...
      this.formErrors = {};
      this.formData.numberOfHosts = undefined;
      this.showBillingForm = false;
      this.numberOfHostsQuoted = undefined;
      // When the input field has been rendered back into existence, focus it for our friendly user.
      await this.forceRender();
      this.$focus('[purpose="quote-input"]');
    },


  }
});

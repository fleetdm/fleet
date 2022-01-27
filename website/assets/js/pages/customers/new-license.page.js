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
    },

    // Syncing / loading state
    syncing: false,

    // Server error state
    cloudError: '',


    quotedPrice: undefined,
    numberOfHostsQuoted: undefined,
    // Success state when the billing form has been submitted
    showBillingForm: false,
    showQuotedPrice: false,
    showSuccessMessage: false,
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

    submittedPaymentForm: async function() {
      // After payment is submitted, Display a success message and let them navigate to the dashboard
      this.showSuccessMessage = true;
      await this.forceRender();
      this.$focus('[purpose="submit-button"]');
    },

    clickGoToDashboard: async function() {
      this.syncing = true;
      window.location = '/customers/dashboard';
    },

    submittedQuoteForm: async function(quote) {
      this.showQuotedPrice = true;
      this.quotedPrice = quote.quotedPrice;
      this.numberOfHostsQuoted = quote.numberOfHosts;
      if(quote.numberOfHosts <= 100) {
        this.formData.quoteId = quote.id;
        this.showBillingForm = true;
      }
      // When the final submit has been rendered into existence, focus it for our friendly user.
      await this.forceRender();
      this.$focus('[purpose="submit-button"]');
    },

    clickScheduleDemo: async function() {
      this.syncing = true;
      // Note: we keep loading spinner present indefinitely so that it is apparent that a new page is loading
      window.location = 'https://calendly.com/fleetdm/demo';
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

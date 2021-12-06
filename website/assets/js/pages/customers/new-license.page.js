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
    // Form rules
    formRules: {

    },

    // Syncing / loading state
    syncing: false,

    // Server error state
    cloudError: '',

    // Success state when form has been submitted
    cloudSuccess: false,
    showBillingForm: false,
    quotedPrice: undefined,
    quotedId: undefined,
    numberOfHostsQuoted: undefined,
    thirtyDaysFromTodayInMS: Date.now() + 30*24*60*60*1000,
    // stripePaymentOptions: {},
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

  },
  mounted: async function() {
    //…

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    submittedPaymentForm: async function() {
      // After payment is submitted, take the user to their dashboard
      this.syncing = true;
      window.location = '/customers/dashboard';
    },

    submittedQuoteForm: async function(quote) {

      if(quote.numberOfHosts > 100) {
        let today = new Date(Date.now());
        this.syncing = true;
        window.location = 'https://calendly.com/fleetdm/demo?month='+today.getFullYear()+'-'+today.getMonth();
        // note: we keep loading spinner present indefinitely so that it is apparent that a new page is loading
      } else {
        this.numberOfHostsQuoted = quote.numberOfHosts;
        this.quotedId = quote.id;
        this.quotedPrice = quote.quotedPrice;
        this.showBillingForm = true;
      }

    },

    clickResetForm: async function() {
      this.formData = {};
      this.showBillingForm = false;
    },

    handleSubmittingPayment: async function() {
      this.syncing = true;
      let paymentArgins = {
        userId: this.me.id,
        quoteId: this.quotedId,
        newPaymentSource: this.formData.paymentSource,
      };
      await Cloud.saveBillingInfoAndSubscribe.with(paymentArgins);
      this.cloudSuccess = true;
    }

  }
});

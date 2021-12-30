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
    showQuotedPrice: false,
    quotedPrice: undefined,
    numberOfHostsQuoted: undefined,
    thirtyDaysFromTodayInMS: Date.now() + 30*24*60*60*1000,
    calendlyLink: '',
    orderComplete: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    let today = new Date(Date.now());
    this.calendlyLink = 'https://calendly.com/fleetdm/demo?month='+today.getFullYear()+'-'+today.getMonth();
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
      this.syncing = true;
      this.orderComplete = true;
      this.syncing = false;
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

    },

    clickScheduleDemo: async function() {
      this.syncing = true;
      // Note: we keep loading spinner present indefinitely so that it is apparent that a new page is loading
      window.location = this.calendlyLink;
    },

    clickResetForm: async function() {
      this.formData = {};
      this.formErrors = {};
      this.showBillingForm = false;
      this.showQuotedPrice = false;
    },


  }
});

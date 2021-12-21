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

    // note: we keep loading spinner present indefinitely so that it is apparent that a new page is loading
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
      // After payment is submitted, take the user to their dashboard
      this.syncing = true;
      this.orderComplete = true;
      this.syncing = false;
      // window.location = '/customers/dashboard';
    },
    clickGoToDashboard: async function() {
      // After payment is submitted, take the user to their dashboard
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
        // this.syncing = true;
      }

    },

    clickScheduleDemo: async function() {
      this.syncing = true;
      // note: we keep loading spinner present indefinitely so that it is apparent that a new page is loading
      window.location = this.calendlyLink;
    },

    clickResetForm: async function() {
      this.formData = {};
      this.showBillingForm = false;
      this.showQuotedPrice = false;
    },


  }
});

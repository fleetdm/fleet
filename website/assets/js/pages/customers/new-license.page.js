parasails.registerPage('new-license', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Form data
    newLicenseFormData: {
      // macosHosts: 0,
      // windowsHosts: 0,
      // linuxHosts: 0,
      // iosHosts: 0,
      // androidHosts: 0,
      // otherHosts: 0,
    },
    formData: {},

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    quoteFormRules: {

    },
    quote: {},
    billingFormRules: {
      paymentSource: {required: true},
      selfHostedAcknowledgment: {required: true, is: true},
    },

    checkoutFormRules: {
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
    if(window.location.hash) {
      window.location.hash = '';
    }
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
    handleSubmittingQuoteForm: async function() {
      let totalNumberOfHosts = _.sum(_.values(this.newLicenseFormData));
      if(totalNumberOfHosts === 0) {
        this.formErrors = {numberOfHosts: 'required'};
        return;
      } else {
        let argins = {
          macosHosts: this.newLicenseFormData.macosHosts ? this.newLicenseFormData.macosHosts : 0,
          windowsHosts: this.newLicenseFormData.windowsHosts ? this.newLicenseFormData.windowsHosts : 0,
          linuxHosts: this.newLicenseFormData.linuxHosts ? this.newLicenseFormData.linuxHosts : 0,
          iosHosts: this.newLicenseFormData.iosHosts ? this.newLicenseFormData.iosHosts : 0,
          androidHosts: this.newLicenseFormData.androidHosts ? this.newLicenseFormData.androidHosts : 0,
          otherHosts: this.newLicenseFormData.otherHosts ? this.newLicenseFormData.otherHosts : 0,
        };
        this.quote = await Cloud.createQuote.with(argins);
      }

    },
    handleSubmittingCheckoutForm: async function() {
      let redirectUrl = await Cloud.getStripeCheckoutSessionUrl.with({
        quoteId: this.formData.quoteId
      });
      this.goto(redirectUrl);
    },
    submittedQuoteForm: async function() {
      if(this.formErrors.numberOfHosts){
        return;
      }
      this.showQuotedPrice = true;
      this.quotedPrice = this.quote.quotedPrice;
      // Convert the quoted price into a string that contains commas.
      this.quotedPrice = this.quotedPrice.toLocaleString('en', {useGrouping:true});
      this.numberOfHostsQuoted = this.quote.numberOfHosts;
      if(this.quote.numberOfHosts < 700) {
        this.formData.quoteId = this.quote.id;
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
      this.showBillingForm = false;
      this.numberOfHostsQuoted = undefined;
      this.showQuotedPrice = false;
      // When the input field has been rendered back into existence, focus it for our friendly user.
      await this.forceRender();
      this.$focus('#macosHosts');
    },


  }
});

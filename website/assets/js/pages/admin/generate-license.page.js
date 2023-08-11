parasails.registerPage('generate-license', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Form data
    formData: {},
    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: {},
    // Form rules
    formRules: {
      numberOfHosts: {required: true},
      organization: {required: true},
      expiresAt: {required: true},
    },
    // Syncing / loading state
    syncing: false,
    // Server error state
    cloudError: '',
    generatedLicenseKey: '',
    showResult: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // Get a formatted date string for year from today's date.
    let oneYearFromNowDateString = moment(Date.now() + (365*24*60*60*1000)).format('YYYY-MM-DD');
    // Set the starting value for the validTo input
    this.formData.expiresAt = oneYearFromNowDateString;
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    handleSubmittingForm: async function() {
      let validToDate = new Date(this.formData.expiresAt);
      let licenseExpiresAt = validToDate.getTime();

      this.generatedLicenseKey = await Cloud.buildLicenseKey.with({
        numberOfHosts: this.formData.numberOfHosts,
        organization: this.formData.organization,
        expiresAt: licenseExpiresAt,
        partnerName: this.formData.partnerName,
      });
    },

    submittedQuoteForm: async function() {
      this.syncing = false;
      this.showResult = true;
    },

    clickCopyLicenseKey: function(){
      $('[purpose="copied-notification"]').finish();
      $('[purpose="copied-notification"]').fadeIn(100).delay(2000).fadeOut(500);
      // https://caniuse.com/mdn-api_clipboard_writetext
      navigator.clipboard.writeText(this.generatedLicenseKey);
    },

    clickClearFormFields: async function() {
      this.generatedLicenseKey = '';
      this.showResult = false;
      this.formErrors = {};
      this.formData = {};
      this.formData.validTo = moment(Date.now() + (365*24*60*60*1000)).format('YYYY-MM-DD');
      await this.forceRender();
    }
  }
});

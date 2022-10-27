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
      validTo: {required: true},
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

    // Get an ISO timestamp for a year from now.
    let validToDefaultValue = new Date(Date.now() + (365*24*60*60*1000)).toISOString();
    // Remove everything but the date from the ISO timestamp
    validToDefaultValue = validToDefaultValue.split('T')[0];
    // Set the default value for the validTo input
    this.formData.validTo = validToDefaultValue;
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    handleSubmittingForm: async function(argins) {
      this.syncing = true;
      let validToDate = new Date(this.formData.validTo);
      let validToTimestamp = validToDate.getTime();
      this.generatedLicenseKey = await Cloud.generateLicenseKey.with({
        numberOfHosts: this.formData.numberOfHosts,
        organization: this.formData.organization,
        validTo: validToTimestamp
      });
    },

    submittedQuoteForm: async function(quote) {
      this.syncing = false;
      this.showResult = true;
    },

    clickCopyLicenseKey: function(){
      $('[purpose="copied-notification"]').finish();
      $('[purpose="copied-notification"]').fadeIn(100).delay(2000).fadeOut(500);
      // https://caniuse.com/mdn-api_clipboard_writetext
      navigator.clipboard.writeText(this.generatedLicenseKey);
    }
  }
});

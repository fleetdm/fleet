parasails.registerPage('patch-faster', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    syncing: false,
    formData: { /* … */ },
    formErrors: { /* … */ },
    cloudError: '',
    cloudSuccess: false,
    demoFormRules: {
      emailAddress: { isEmail: true, required: true },
      firstName: { required: true },
      lastName: { required: true },
      numberOfHosts: { required: true },
    },
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

    handleSubmittingDemoForm: async function(argins) {
      this.syncing = true;
      // Supply fields not collected from the form.
      argins.organization = argins.organization || '';
      argins.primaryBuyingSituation = 'it-misc';
      if(typeof window.lintrk !== 'undefined') {
        window.lintrk('track', { conversion_id: 18587089 });// eslint-disable-line camelcase
      }
      let report = await Cloud.deliverTalkToUsFormSubmission.with(argins);
      if(report.icp) {
        if(typeof gtag !== 'undefined') { gtag('event', 'fleet_website__contact_forms__demo__icp'); }
        if(typeof window.lintrk !== 'undefined') { window.lintrk('track', { conversion_id: 27493081 }); }// eslint-disable-line camelcase
      } else {
        if(typeof gtag !== 'undefined') { gtag('event', 'fleet_website__contact_forms__demo'); }
      }
      if(typeof gtag !== 'undefined') {
        gtag('event', 'conversion', {
          'send_to': 'AW-10788733823/aNrhCNSYrPobEP-GvJgo',
          'value': 1.0,
          'currency': 'USD'
        });
      }
      if(typeof qualified !== 'undefined') {
        qualified('saveFormData', {
          email: this.formData.emailAddress,
          name: this.formData.firstName,
          company: this.formData.organization,
        });
        qualified('showFormExperience', 'experience-1772126772950');
      }
      this.goto(report.eventUrl);
    },

  }
});

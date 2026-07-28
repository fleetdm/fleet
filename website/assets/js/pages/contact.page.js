parasails.registerPage('contact', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    formToDisplay: 'talk-to-us',
    primaryBuyingSituation: undefined,
    // Main syncing/loading state for this page.
    syncing: false,

    // Form data
    formData: { /* … */ },

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    // "talk to us" Form rules
    talkToUsFormRules: {
      emailAddress: {isEmail: true, required: true},
      firstName: {required: true},
      lastName: {required: true},
      organization: {required: true},
      numberOfHosts: {required: true},
    },
    // Contact form rules
    contactFormRules: {
      emailAddress: {isEmail: true, required: true},
      firstName: {required: true},
      lastName: {required: true},
      message: {required: true},
    },
    // Application form rules
    applicationFormRules: {
      emailAddress: {isEmail: true, required: true},
      firstName: {required: true},
      lastName: {required: true},
      linkedinProfileUrl: {required: true},
      position: {required: true},
      location: {required: true},
      message: {required: true},
    },

    workshopRequestFormRules: {
      emailAddress: {isEmail: true, required: true},
      firstName: {required: true},
      lastName: {required: true},
      location: {required: true},
      numberOfHosts: {required: true},
      managedPlatforms: {
        required: true,
        custom: (selectedPlatforms)=>{
          return _.keysIn(selectedPlatforms).length > 0 && _.contains(_.values(selectedPlatforms), true);
        }
      },

    },

    formDataToPrefillForLoggedInUsers: {},

    // Server error state for the form
    cloudError: '',

    // Success state when form has been submitted
    cloudSuccess: false,

    // For personalizing the message at the top of the contact form for logged-in users.
    psychologicalStage: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(this.formToShow === 'contact'){
      this.formToDisplay = this.formToShow;
    } else if(!this.primaryBuyingSituation){
      // Otherwise, default to the formToShow value from the page's controller.
      this.formToDisplay = this.formToShow;
    }

    if(this.me){// prefill from database
      this.formDataToPrefillForLoggedInUsers.emailAddress = this.me.emailAddress;
      this.formDataToPrefillForLoggedInUsers.firstName = this.me.firstName;
      this.formDataToPrefillForLoggedInUsers.lastName = this.me.lastName;
      this.formDataToPrefillForLoggedInUsers.organization = this.me.organization;
      this.formData = _.clone(this.formDataToPrefillForLoggedInUsers);
      this.psychologicalStage = this.me.psychologicalStage;
    }

    if (window.location.hash === '#message') {// prefill from URL bar
      this.formToDisplay = 'contact';
    }
    if (window.location.hash === '#apply') {// prefill from URL bar
      this.formToDisplay = 'apply';
    }
    if (window.location.hash === '#gitops') {// prefill from URL bar
      this.formToDisplay = 'gitops-workshop-request';
      this.formData.managedPlatforms = {};
    }
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    submittedContactForm: async function() {
      if(typeof gtag !== 'undefined'){
        gtag('event','fleet_website__contact_forms');
        // Additional conversion tracking
        gtag('event', 'conversion', {
          'send_to': 'AW-10788733823/e9FrCNGYrPobEP-GvJgo',
          'value': 1.0,
          'currency': 'USD'
        });
      }
      if(typeof window.lintrk !== 'undefined') {
        window.lintrk('track', { conversion_id: 18587089 });// eslint-disable-line camelcase
      }
      if(typeof qualified !== 'undefined') {
        qualified('saveFormData',
        {
          email: this.formData.emailAddress,
          name: this.formData.firstName +' '+ this.formData.lastName,

        });
        qualified('showFormExperience', 'experience-1772126772950');
      }

      // Show the success message.
      this.cloudSuccess = true;

    },
    handleSubmittingTalkToUsForm: async function(argins) {
      this.syncing = true;
      if(typeof window.lintrk !== 'undefined') { window.lintrk('track', { conversion_id: 18587089 }); }// eslint-disable-line camelcase
      let report = await Cloud.deliverTalkToUsFormSubmission.with(argins);

      // Look at result from talking to api and decide what event to track.
      if(report.icp){
        if(typeof gtag !== 'undefined'){ gtag('event','fleet_website__contact_forms__demo__icp'); }
        if(typeof window.lintrk !== 'undefined') { window.lintrk('track', { conversion_id: 27493081 }); } // eslint-disable-line camelcase
      } else {
        if(typeof gtag !== 'undefined'){ gtag('event','fleet_website__contact_forms__demo'); }
      }

      // Additional conversion tracking
      // =>Mike: Why is this here?  I checked a blame and I see that we added this based on a PDF we received from a vendor, but couldn't see why from the linked issue.
      if(typeof gtag !== 'undefined'){
        gtag('event', 'conversion', {
          'send_to': 'AW-10788733823/aNrhCNSYrPobEP-GvJgo',
          'value': 1.0,
          'currency': 'USD'
        });
      }//ﬁ

      if(typeof qualified !== 'undefined') {
        qualified('saveFormData',
        {
          email: this.formData.emailAddress,
          name: this.formData.firstName +' '+ this.formData.lastName,
          company: this.formData.organization,
          how_many_hostsdevices_do_you_want_to_manage: this.formData.numberOfHosts,// eslint-disable-line camelcase
        });
        qualified('showFormExperience', 'experience-1772126772950');
      }

      this.goto(report.eventUrl);
    },

    submittedApplicationForm: async function() {
      // Show the success message.
      this.cloudSuccess = true;
    },

    submittedWorkshopRequestForm: async function() {
      if(typeof gtag !== 'undefined'){
        // Additional conversion tracking
        gtag('event', 'conversion', {
          'send_to': 'AW-10788733823/RULOCNeYrPobEP-GvJgo',
          'value': 1.0,
          'currency': 'USD'
        });
      }
      if(typeof qualified !== 'undefined') {
        qualified('saveFormData',
        {
          email: this.formData.emailAddress,
          name: this.formData.firstName +' '+ this.formData.lastName,
          company: this.formData.organization,
          how_many_hostsdevices_do_you_want_to_manage: this.formData.numberOfHosts,// eslint-disable-line camelcase
        });
        qualified('showFormExperience', 'experience-1772126772950');
      }
      // Show the success message.
      this.cloudSuccess = true;
    },

    clickSelectCustomCheckbox: async function() {
      await this.forceRender();
    },

    clickSwitchForms: function(form) {
      if(this.me){
        this.formData = _.clone(this.formDataToPrefillForLoggedInUsers);
      } else {
        this.formData = {};
      }
      this.formErrors = {};
      this.cloudError = '';
      this.formToDisplay = form;
    }

  }
});

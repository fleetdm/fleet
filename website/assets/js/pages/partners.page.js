parasails.registerPage('partners', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    isSafariThirteen: bowser.safari && _.startsWith(bowser.version, '13'),
    isIosThirteen: bowser.safari && _.startsWith(bowser.version, '13') && bowser.ios,

    // For form modals
    modal: '',

    // For partner registration form
    partnerFormData: {
      partnerType: 'reseller',
      servicesOffered: {},
    },
    partnerFormRules: {
      submittersFirstName: { required: true },
      submittersLastName: { required: true },
      submittersEmailAddress: { required: true },
      submittersOrganization: { required: true },
      partnerType: { required: true },
      partnerWebsite: { required: true },
      partnerCountry: { required: true },
      notes: {required: true },
      // IF partnerType === reseller
      // servicesOffered: {required: true,},
      // numberOfHosts: { required: true },

      // IF partnerType === integrations
      // servicesCategory: { required: true },
    },

    // For deal registration form
    dealRegistrationFormData: {
      platforms: {},
      useCase: {},
    },
    dealRegistrationFormRules: {
      submittersFirstName: { required: true },
      submittersLastName: { required: true },
      submittersEmailAddress: { required: true, isEmail: true },
      submittersOrganization: { required: true },
      submitterIsExistingPartner: { required: true },
      customersOrganization: { required: true },
      customersName: { required: true },
      customersEmailAddress: { required: true, isEmail: true },
      dealStage: {required: true},
      expectedClose: { required: true },
      numberOfHosts: { required: true },
      platforms: {
        required: true,
        custom: (platforms)=>{
          return _.keysIn(platforms).length > 0 && _.contains(_.values(platforms), true);
        }
      },
      useCase: {
        required: true,
        custom: (useCase)=>{
          return _.keysIn(useCase).length > 0 && _.contains(_.values(useCase), true);
        }
      },
      notes: { required: true },
    },


    // used by all forms.
    formErrors: {},
    syncing: false,
    cloudError: undefined,
    cloudSuccess: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    if(window.location.hash){
      if(window.location.hash === '#deals') {
        this.modal = 'deal-registration';
      }
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickOpenModal: function(modalName) {
      this.modal = modalName;
    },
    clickOpenPartnerModal: function(partnerType) {
      this.partnerFormData.partnerType = partnerType;
      this.modal = 'partner';
    },
    clickSelectCustomCheckbox: async function() {
      await this.forceRender();
    },

    clickResetForm: function() {
      if(this.modal === 'deal-registration') {
        this.dealRegistrationFormData = {
          platforms: {},
          useCase: {},
        };
      }
      if(this.modal === 'partner') {
        this.partnerFormData = {
          partnerType: 'reseller',
          servicesOffered: {},
        };
      }
      this.formErrors = {};
    },


    handleSubmittingPartnerForm: async function(argins) {
      this.syncing = true;
      // check to make sure that we have the required values depending on selected partnerType value.
      if(argins.partnerType === 'reseller') {
        let servicesOffered = argins.servicesOffered;
        // if this was the form for resellers and no services were selected, add a formError for this input.
        if(!_.contains(_.values(servicesOffered), true)){
          this.formErrors.servicesOffered = {required: true};
        }
        if(!argins.numberOfHosts) {
          this.formErrors.numberOfHosts = {required: true};
        }
      } else if(argins.partnerType === 'integrations') {
        if(!argins.servicesCategory){
          this.formErrors.servicesCategory = {required: true};
        }
      }

      await Cloud.deliverPartnerRegistrationSubmission.with(argins).tolerate((err)=>{
        this.syncing = false;
        this.cloudError = err;
      });
    },



    closeModal: function() {
      this.clickResetForm();
      this.modal = undefined;
      this.cloudSuccess = false;
      this.dealRegistrationFormData = {
        platforms: {},
        useCase: {},
      };
      this.partnerFormData = {
        partnerType: 'reseller',
        servicesOffered: {},
      };
    },
    submittedDealForm: function() {
      if(!this.cloudError) {
        this.cloudSuccess = true;
      }
    },
    submittedPartnerForm: function() {
      if(!this.cloudError) {
        this.cloudSuccess = true;
      }
    },
  }
});

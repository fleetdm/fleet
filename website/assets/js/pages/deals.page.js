parasails.registerPage('deals', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Form data
    formData: {
      submittersOrganization: 'CDW',
    },

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    // Form rules
    formRules: {
      submittersFirstName: { required: true },
      submittersLastName: { required: true },
      submittersEmailAddress: { required: true, isEmail: true },
      submittersOrganization: { required: true },
      customersFirstName: { required: true },
      customersLastName: { required: true },
      customersEmailAddress: { required: true, isEmail: true },
      customersOrganization: { required: true },
      expectedDealSize: { required: true },
      expectedCloseDate: { required: true },
    },
    // Syncing / loading state
    syncing: false,
    // Server error state
    cloudError: '',
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
    submittedForm: async function() {
      this.syncing = false;
      this.showSuccessMessage = true;
    },
    clickResetForm: async function() {
      this.showSuccessMessage = false;
      this.cloudError = '';
      this.formErrors = {};
      this.formData = {
        submittersOrganization: 'CDW',
      };
      await this.forceRender();
    },

    clickChoosePreferredHosting: async function(value){
      this.formData.preferredHosting = value;
      await this.forceRender();
    },

    typeClearOneFormError: async function(field) {
      if(this.formErrors[field]){
        this.formErrors = _.omit(this.formErrors, field);
      }
    },
  }
});

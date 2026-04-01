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
    partnerFormRules: {},

    // For deal registration form
    dealRegistrationFormData: {},
    dealRegistrationFormRules: {},



    // used by all forms.
    formErrors: {},
    syncing: false,
    cloudError: undefined,
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

    closeModal: function() {
      this.modal = undefined;
    },
    submittedPartnerForm: function() {

    },
  }
});

parasails.registerPage('connect-vanta', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    //…
    formData: { },
    formErrors: { },
    formRules: {
      emailAddress: {required: true, isEmail: true},
      fleetInstanceUrl: {required: true, custom:(value)=>{
        return !! _.startsWith(value, 'https://') || _.startsWith(value, 'http://');
      }},
      fleetApiKey: {required: true},
    },
    syncing: false,
    cloudError: '',
    vantaAuthorizationRequestURL: '',
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

    handleSubmittingAuthorizationForm: async function(argins) {
      this.vantaAuthorizationRequestURL = await Cloud.createVantaAuthorizationRequest.with(argins);
    },

    submittedAuthorizationForm: async function() {
      this.syncing = true;
      this.goto(this.vantaAuthorizationRequestURL);
    },

    clickClearErrors: async function() {
      this.cloudError = '';
      this.formErrors = {};
      await this.forceRender();
    },

  }
});

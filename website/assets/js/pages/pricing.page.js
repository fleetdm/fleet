parasails.registerPage('pricing', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    formData: {},
    estimatedCost: '', // For pricing calculator
    estimatedUltimateCostPerHost: 7,
    displaySecurityPricingMode: false, // For pricing mode switch
    estimatedUltimateCostPerHostHasBeenUpdated: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function(){
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickOpenChatWidget: function() {
      if(window.HubSpotConversations && window.HubSpotConversations.widget){
        window.HubSpotConversations.widget.open();
      }
    },
    updateEstimatedTotal: function() {
      let total =
      (7 * (this.formData.macos ? this.formData.macos : 0)) +
      (7 * (this.formData.windows ? this.formData.windows : 0)) +
      (2 * (this.formData.linux ? this.formData.linux : 0)) +
      (2 * (this.formData.other ? this.formData.other : 0));
      let totalNumberOfDevices =
      (1 * (this.formData.macos ? this.formData.macos : 0)) +
      (1 * (this.formData.windows ? this.formData.windows : 0)) +
      (1 * (this.formData.linux ? this.formData.linux : 0)) +
      (1 * (this.formData.other ? this.formData.other : 0));
      this.estimatedCost = Number(total);
      if(totalNumberOfDevices < 1){
        this.estimatedUltimateCostPerHost = 7;
        this.estimatedUltimateCostPerHostHasBeenUpdated = false;
      } else {
        this.estimatedUltimateCostPerHost = this.estimatedCost / totalNumberOfDevices;
        this.estimatedUltimateCostPerHostHasBeenUpdated = true;
      }

    },
  }
});

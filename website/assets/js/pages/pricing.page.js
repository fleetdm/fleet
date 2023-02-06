parasails.registerPage('pricing', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    formData: {},
    estimatedCost: '', // For pricing calculator
    estimatedUltimateCostPerHost: 7,
    displaySecurityPricingMode: true, // For pricing mode switch
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
    clickChatButton: function() {
      // Temporary hack to open the chat
      // (there's currently no official API for doing this outside of React)
      //
      // > Alex: hey mike! if you're just trying to open the chat on load, we actually have a `defaultIsOpen` field
      // > you can set to `true` :) i haven't added the `Papercups.open` function to the global `Papercups` object yet,
      // > but this is basically what the functions look like if you want to try and just invoke them yourself:
      // > https://github.com/papercups-io/chat-widget/blob/master/src/index.tsx#L4-L6
      // > ~Dec 31, 2020
      window.dispatchEvent(new Event('papercups:open'));
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

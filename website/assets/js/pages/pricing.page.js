parasails.registerPage('pricing', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    displaySecurityPricingMode: false, // For pricing mode switch
    // The order of categories in the pricing table for different pricing modes
    itModeCategoryOrder: [
      'Device management',
      'Support',
      'Inventory management',
      'Collaboration',
      'Security and compliance',
      'Monitoring',
      'Data outputs',
      'Deployment'
    ],
    securityModeCategoryOrder: [
      'Security and compliance',
      'Monitoring',
      'Inventory management',
      'Collaboration',
      'Support',
      'Data outputs',
      'Device management',
      'Deployment'
    ],
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
  watch: {
    displaySecurityPricingMode: function() {
      // When the pricing mode changes, sort the pricing table based on the selected mode.
      if(this.displaySecurityPricingMode){
        this.pricingTable.sort((a, b)=>{
          return this.securityModeCategoryOrder.indexOf(a.categoryName) - this.securityModeCategoryOrder.indexOf(b.categoryName);
        });
      } else {
        this.pricingTable.sort((a, b)=>{
          return this.itModeCategoryOrder.indexOf(a.categoryName) - this.itModeCategoryOrder.indexOf(b.categoryName);
        });
      }
    }
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
  }
});

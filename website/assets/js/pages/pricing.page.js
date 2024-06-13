parasails.registerPage('pricing', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    pricingMode: 'all',
    modal: '',
    selectedFeature: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(this.primaryBuyingSituation){
      if(['eo-security', 'vm'].includes(this.primaryBuyingSituation)){
        this.pricingMode = 'Security';
      } else {
        this.pricingMode = 'IT';
      }
    }
  },
  mounted: async function(){
    // Tooltips for desktop users are opened by a user hovering their cursor over them.
    $('[data-toggle="tooltip"]').tooltip({
      container: '#pricing',
      trigger: 'hover',
    });
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickChangePricingMode: async function(pricingMode){
      this.pricingMode = pricingMode;
    },
    clickOpenMobileTooltip: function(feature){
      this.selectedFeature = feature;
      this.modal = 'mobileTooltip';
    },
    closeModal: function() {
      this.selectedFeature = undefined;
      this.modal = '';
    }
  }
});

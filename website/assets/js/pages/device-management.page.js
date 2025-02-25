parasails.registerPage('device-management-page', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    modal: '',
    comparisonMode: 'omnissa'
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    $('[data-toggle="tooltip"]').tooltip({
      container: '#device-management-page',
      trigger: 'hover',
    });
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickOpenVideoModal: function(modalName) {
      this.modal = modalName;
    },
    closeModal: function() {
      this.modal = undefined;
    },
    clickSwagRequestCTA: function () {
      if(window.gtag !== undefined){
        gtag('event','fleet_website__swag_request');
      }
      if(window.lintrk !== undefined) {
        window.lintrk('track', { conversion_id: 18587105 });// eslint-disable-line camelcase
      }
      if(window.analytics !== undefined) {
        analytics.track('fleet_website__swag_request');
      }
      this.goto('https://kqphpqst851.typeform.com/to/ZfA3sOu0#from_page=device-managment');
    },
  }
});

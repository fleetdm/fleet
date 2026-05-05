parasails.registerPage('fleet-premium-trial', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    //…
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
    closeModalAndRedirect: function() {
      if(window.navigation && window.navigation.canGoBack){
        if(window.navigation.entries()){
          let recentNavigationEntries = window.navigation.entries();
          if(recentNavigationEntries && ['http://localhost:2024/login', 'https://fleetdm.com/login'].includes(recentNavigationEntries[0].url)) {
            this.goto('/');
          }
        }
        window.navigation.back();
      } else {
        this.goto('/');
      }
    },
    clickCopyLicenseKey: async function() {
      $('[purpose="command-copy-button"]').addClass('copied');
      await setTimeout(()=>{
        $('[purpose="command-copy-button"]').removeClass('copied');
      }, 2000);
      navigator.clipboard.writeText(this.trialLicenseKey);
    }
  }
});

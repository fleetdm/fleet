parasails.registerPage('transparency', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    //…
    showSecureframeBanner: false,
    modal: undefined,
    isDesktopUser: true,
    guessedPlatformName: 'computer',// Default to computer.
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    if(bowser.ios && !bowser.ipad) {
      this.isDesktopUser = false;
      this.guessedPlatformName = 'iPhone';
    } else if(bowser.ipad || (navigator.maxTouchPoints > 1 && bowser.mac)) {
      this.isDesktopUser = false;
      this.guessedPlatformName = 'iPad';
    } else if(bowser.android) {
      this.isDesktopUser = false;
      this.guessedPlatformName = 'Android device';
    }
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
  }
});

parasails.registerPage('homepage', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    modal: undefined,
    selectedCategory: 'mdm',
    formData: { /* … */ },
    formErrors: { /* … */ },

    // Form rules
    formRules: {
      emailAddress: {isEmail: true, required: true},
    },

    syncing: false,

    // Server error state for the form
    cloudError: '',
    cloudSuccess: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
    if(window.location.hash === '#unsubscribed'){
      this.modal = 'unsubscribed';
      window.location.hash = '';
    }
  },
  mounted: async function() {
    await setTimeout(()=>{
      this.animateHeroTicker();

    }, 1200)
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    animateHeroTicker: function() {
      setInterval(()=>{
        let currentTickerOption = $('[purpose="hero-ticker-option"].visible');

        if (currentTickerOption.length === 0) {
          currentTickerOption = $('[purpose="hero-ticker-option"]').first();
          currentTickerOption.addClass('visible');
          return;
        }
        let nextTickerOption = currentTickerOption.nextAll('[purpose="hero-ticker-option"]').first();
        if (nextTickerOption.length === 0) {
          nextTickerOption = $('span[purpose="hero-ticker-option"]').first();
        }
        currentTickerOption.removeClass('visible');
        nextTickerOption.addClass('visible');
      }, 1200);
    },
    clickOpenVideoModal: function(modalName) {
      this.modal = modalName;
    },

    closeModal: function() {
      this.modal = undefined;
    },
    submittedNewsletterForm: async function() {
      // Show the success message.
      this.cloudSuccess = true;
      this.formData = {};
      await setTimeout(()=>{
        this.cloudSuccess = false;
      }, 10000);
    },
  }
});

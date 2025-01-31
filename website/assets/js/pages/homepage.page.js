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
    animationDelayInMs: 1200,
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
    this.animateHeroTicker();
    if(this.primaryBuyingSituation && !['vm', 'eo-security'].includes(this.primaryBuyingSituation)){
      this.animateBottomTicker();
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    animateHeroTicker: function() {
      setInterval(()=>{
        let currentTickerOption = $('[purpose="hero-ticker-option"].visible');
        if(currentTickerOption) {
          // [?]:https://api.jquery.com/nextAll/#nextAll-selector
          let nextTickerOption = currentTickerOption.nextAll('[purpose="hero-ticker-option"]').first();
          currentTickerOption.removeClass('visible').addClass('animating-out');
          nextTickerOption.addClass('visible');
          setTimeout(()=>{
            currentTickerOption.removeClass('animating-out');
          }, 1000);
        }
      }, this.animationDelayInMs);
    },
    animateBottomTicker: function() {
      setInterval(()=>{
        let currentTickerOption = $('[purpose="bottom-cta-ticker-option"].visible');
        if(currentTickerOption) {
          // [?]:https://api.jquery.com/nextAll/#nextAll-selector
          let nextTickerOption = currentTickerOption.nextAll('[purpose="bottom-cta-ticker-option"]').first();
          currentTickerOption.removeClass('visible').addClass('animating-out');
          nextTickerOption.addClass('visible');
          setTimeout(()=>{
            currentTickerOption.removeClass('animating-out');
          }, 1000);
        }
      }, this.animationDelayInMs);
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

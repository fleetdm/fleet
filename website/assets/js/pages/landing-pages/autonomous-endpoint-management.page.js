parasails.registerPage('autonomous-endpoint-management', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    modal: '',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    this.animateTicker();
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    animateTicker: function() {
      setInterval(()=>{
        let currentTickerOption = $('[purpose="ticker-option"].visible');
        if(currentTickerOption) {
          if (currentTickerOption.length === 0) {
            currentTickerOption = $('[purpose="ticker-option"]').first();
            currentTickerOption.addClass('visible');
            return;
          }
          // [?]:https://api.jquery.com/nextAll/#nextAll-selector
          let nextTickerOption = currentTickerOption.nextAll('[purpose="ticker-option"]').first();
          // If we've reached the end of the list, pick the first option to be the next ticker option
          if (nextTickerOption.length === 0) {
            nextTickerOption = $('span[purpose="ticker-option"]').first();
          }
          currentTickerOption.removeClass('visible').addClass('animating-out');
          nextTickerOption.addClass('visible');
          setTimeout(()=>{
            currentTickerOption.removeClass('animating-out');
          }, 1000);//œ (note this 1000 had better be less than the 1200 below)
        }//ﬁ
      }, 1200);//œ
    },
    clickOpenVideoModal: function(modalName) {
      this.modal = modalName;
    },
    closeModal: function() {
      this.modal = undefined;
    },
  }
});

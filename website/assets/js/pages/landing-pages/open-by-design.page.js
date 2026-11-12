parasails.registerPage('open-by-design', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    modal: undefined,
    animationDelayInMs: 2000,

    // For MDM comparison table
    comparisonModeForIt: 'intune',
    comparisonModeFriendlyNames: {
      intune: 'Intune',
      jamf: 'Jamf Pro',
      omnissa: 'Workspace ONE',
      ivanti: 'Ivanti',
      tanium: 'Tanium',
      ansible: 'Ansible',
      puppet: 'Puppet',
      chef: 'Chef',
      rapid: 'Rapid 7',
      crowdstrike: 'Crowdstrike',
      qualys: 'Qualys',
      tenable: 'Tenable',
      defender: 'Defender',
      patchmypc: 'PatchMyPC',
    }
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    this.animateTicker('hero-ticker-option');
    $('[data-toggle="tooltip"]').tooltip({
      container: '#open-by-design',
      trigger: 'hover',
    });
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    animateTicker: function(tickerOptionPurpose) {
      let tickerOptionSelector = `[purpose="${tickerOptionPurpose}"]`;
      setInterval(()=>{
        let currentTickerOption = $(`${tickerOptionSelector}.visible`);
        if (currentTickerOption.length === 0) {
          $(tickerOptionSelector).first().addClass('visible');
          return;
        }
        // [?]:https://api.jquery.com/nextAll/#nextAll-selector
        let nextTickerOption = currentTickerOption.nextAll(tickerOptionSelector).first();
        // If we've reached the end of the list, pick the first option to be the next ticker option
        if (nextTickerOption.length === 0) {
          nextTickerOption = $(tickerOptionSelector).first();
        }
        currentTickerOption.removeClass('visible').addClass('animating-out');
        nextTickerOption.addClass('visible');
        setTimeout(()=>{
          currentTickerOption.removeClass('animating-out');
        }, 1000);
      }, this.animationDelayInMs);
    },
    clickSwitchComparisonTableColumn: async function(option){
      this.comparisonModeForIt = option;
      await setTimeout(()=>{
        $('[data-toggle="tooltip"]').tooltip({
          container: '#open-by-design',
          trigger: 'hover',
        });
      }, 250);
    },
    closeModal: function() {
      this.modal = undefined;
    },
  }
});

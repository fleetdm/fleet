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
    animationDelayInMs: 2000,
    syncing: false,

    // Server error state for the form
    cloudError: '',
    cloudSuccess: false,

    // For MDM comparison table
    comparisonModeForIt: 'jamf',
    comparisonModeFriendlyNames: {
      jamf: 'Jamf Pro',
      sccm: 'SCCM',
      omnissa: 'Workspace ONE',
      intune: 'Intune',
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
    if(window.location.hash === '#unsubscribed'){
      this.modal = 'unsubscribed';
      window.location.hash = '';
    }
  },
  mounted: async function() {
    this.animateTicker('hero-ticker-option');
    this.animateTicker('bottom-cta-ticker-option');
    $('[data-toggle="tooltip"]').tooltip({
      container: '#homepage',
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
    clickOpenVideoModal: function(modalName) {
      this.modal = modalName;
    },
    clickSwitchComparisonTableColumn: async function(option){
      this.comparisonModeForIt = option;
      await setTimeout(()=>{
        $('[data-toggle="tooltip"]').tooltip({
          container: '#homepage',
          trigger: 'hover',
        });
      }, 250);
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

parasails.registerPage('configuration-builder', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedPlatform: undefined,
    step: 'platform-select',
    formErrors: {},
    platformSelectFormData: {
      platform: undefined,
    },
    platformSelectFormRules: {
      platform: {required: true},
    },
    syncing: false,
    cloudError: undefined,
    searchKeyword: undefined,

    // For configuration builder form
    //
    expandedCategory: undefined,
    selectedSubcategory: undefined,
    selectedPayloadOptions: {},
    payloadOptionsToDisplayGroupedByCategory: {},
    payloadOptionsToDisplay: [],
    // TODO: build this in the view action from some sort of configuration file.
    configurationCategories: [
      {
        name: 'Privacy & security',
        subcategories: [
          {
            name: 'Device lock',
            description: 'Settings related to screen lock and passwords.',
            learnMoreLink: '/tables/screenlock#apple',
            payloadOptions: [
              {
                name: 'Max inactivity time before device locks',
                uniqueSlug: 'windows-device-lock-max-inactivity',
                category: 'Device lock',
                tooltip: 'The number of seconds a device can remain inactive before a password is required to unlock the device.',
                supportedAccessTypes: ['add', 'replace'],
                acceptedValue: {
                  type: 'number',
                  maxValue: '9000',
                  minValue: '1',
                }
              }
            ]
          }
        ]
      },
    ],
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    $('[data-toggle="tooltip"]').tooltip({
      container: '#configuration-builder',
      trigger: 'hover',
    });
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickGotoNextStep: function() {
      if(this.selectedPlatform){
        this.step === 'configuration-builder';
      }
    },
    handleSubmittingPlatformSelectForm: async function() {
      this.selectedPlatform = this.platformSelectFormData.platform;
      this.step = 'configuration-builder';
    },
    typeFilterSettings: async function() {
      // TODO.
    },
    clickExpandCategory: function(category) {
      this.expandedCategory = category;
    },
    clickSelectSubcategory: function(subcategory) {
      this.selectedSubcategory = subcategory;

    },

    clickSelectPayloadOption: function(payloadOption) {
      if(_.find(this.payloadOptionsToDisplay, {name: payloadOption.name})) {
        this.payloadOptionsToDisplay = _.without(this.payloadOptionsToDisplay, payloadOption);
      } else {
        this.payloadOptionsToDisplay.push(payloadOption);
      }
      this.payloadOptionsToDisplayGroupedByCategory = _.groupBy(this.payloadOptionsToDisplay, 'category');
      console.log(this.payloadOptionsToDisplayGroupedByCategory);
    },
    clickRemoveAllPayloadOptions: function() {
      this.selectedPayloadOptions = {};
      this.payloadOptionsToDisplay = [];
      this.payloadOptionsToDisplayGroupedByCategory = {};
    },
    clickRemovePayloadOption: function(option) {
      this.payloadOptionsToDisplay = _.without(this.payloadOptionsToDisplay, option);
      this.payloadOptionsToDisplayGroupedByCategory = _.groupBy(this.payloadOptionsToDisplay, 'category');
      this.selectedPayloadOptions[option.uniqueSlug] = false;
    }
  }
});

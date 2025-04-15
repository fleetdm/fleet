parasails.registerPage('configuration-builder', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedPlatform: undefined,
    // step: 'platform-select',
    step: 'configuration-builder',
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
    configurationProfileFormData: {},
    // TODO: build this based on what options are selected. (Can probably just start with {required: true} as a rule for each added option.)
    configurationProfileFormRules: {},
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
              },
              {
                name: 'Require alphanumeric device password',
                uniqueSlug: 'windows-device-lock-require-alphanumeric-device-password',
                category: 'Device lock',
                supportedAccessTypes: ['add', 'replace'],
                acceptedValue: {
                  type: 'radio',
                  options: [
                    {
                      name: 'Password or alphanumeric PIN required',
                      value: '1'
                    },
                    {
                      name: 'Password or Numeric PIN required',
                      value: '2'
                    },
                    {
                      name: 'Password, Numeric PIN, or alphanumeric PIN required',
                      value: '3',
                    }
                  ]
                }
              },
              {
                name: 'Enable device password',
                uniqueSlug: 'windows-device-lock-enable-device-password',
                tooltip: 'Require a password to unlock the device',
                category: 'Device lock',
                supportedAccessTypes: ['add', 'replace'],
                acceptedValue: {
                  type: 'boolean',
                }
              }
            ]
          }
        ]
      },
      // {
      //   name: 'Second category',
      //   subcategories: [
      //     {
      //       name: 'This is the same as the other one.',
      //       description: 'Settings related to screen lock and passwords.',
      //       learnMoreLink: '/tables/screenlock#apple',
      //       payloadOptions: [
      //         {
      //           name: 'Max inactivity time before device locks',
      //           uniqueSlug: 'windows-device-lock-max-inactivity-2',
      //           category: 'Device lock',
      //           tooltip: 'The number of seconds a device can remain inactive before a password is required to unlock the device.',
      //           supportedAccessTypes: ['add', 'replace'],
      //           acceptedValue: {
      //             type: 'number',
      //             maxValue: '9000',
      //             minValue: '1',
      //           }
      //         },
      //         {
      //           name: 'Require alphanumeric device password',
      //           uniqueSlug: 'windows-device-lock-require-alphanumeric-device-password-2',
      //           category: 'Device lock',
      //           supportedAccessTypes: ['add', 'replace'],
      //           acceptedValue: {
      //             type: 'radio',
      //             options: [
      //               {
      //                 name: 'Password or alphanumeric PIN required',
      //                 value: '1'
      //               },
      //               {
      //                 name: 'Password or Numeric PIN required',
      //                 value: '2'
      //               },
      //               {
      //                 name: 'Password, Numeric PIN, or alphanumeric PIN required',
      //                 value: '3',
      //               }
      //             ]
      //           }
      //         },
      //         {
      //           name: 'Enable device password',
      //           uniqueSlug: 'windows-device-lock-enable-device-password-2',
      //           tooltip: 'Require a password to unlock the device',
      //           category: 'Device lock',
      //           supportedAccessTypes: ['add', 'replace'],
      //           acceptedValue: {
      //             type: 'boolean',
      //           }
      //         }
      //       ]
      //     }
      //   ]
      // },
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
      $('[data-toggle="tooltip"]').tooltip({
        container: '#configuration-builder',
        trigger: 'hover',
      });
    },

    clickSelectPayloadOption: function(payloadOption) {
      if(this.selectedPayloadOptions[payloadOption.uniqueSlug]){
        this.payloadOptionsToDisplay = _.without(this.payloadOptionsToDisplay, payloadOption);
        this.configurationProfileFormRules = _.omit(this.configurationProfileFormRules, payloadOption.uniqueSlug+'-value');
        this.configurationProfileFormRules = _.omit(this.configurationProfileFormRules, payloadOption.uniqueSlug+'-access-type');
        this.configurationProfileFormData = _.omit(this.configurationProfileFormData, payloadOption.uniqueSlug+'-access-type');
        this.configurationProfileFormData = _.omit(this.configurationProfileFormData, payloadOption.uniqueSlug+'-value');
      } else {
        this.payloadOptionsToDisplay.push(payloadOption);
        this.configurationProfileFormRules[payloadOption.uniqueSlug+'-value'] = {required: true};
        this.configurationProfileFormRules[payloadOption.uniqueSlug+'-access-type'] = {required: true};
      }
      this.payloadOptionsToDisplayGroupedByCategory = _.groupBy(this.payloadOptionsToDisplay, 'category');
      // Depending on the payload option's acceptedValue.type value, we'll update the form rules and formData for the configuration builder form.
      if(payloadOption.acceptedValue.type === 'boolean') {
        this.configurationProfileFormData[payloadOption.uniqueSlug+'-value'] = false;
      }

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
    },
    handleSubmittingConfigurationBuilderForm: function() {
      console.log(this.configurationProfileFormData);
    },
    clickClearOneFormError: function(field) {
      if(this.formErrors[field]){
        this.formErrors = _.omit(this.formErrors, field);
      }
    },
  }
});

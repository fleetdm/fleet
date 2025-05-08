parasails.registerPage('configuration-builder', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // selectedPlatform: undefined,
    // step: 'platform-select',
    selectedPlatform: 'windows',
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
    // modal: 'download-profile',
    // modal: 'multiple-payloads-selected',
    modal: undefined,
    expandedCategory: undefined,
    selectedSubcategory: undefined,

    // TODO: build this in the view action from some sort of configuration file.
    // configurationCategories: [
    //   {
    //     name: 'Privacy & security',
    //     subcategories: [
    //       {
    //         name: 'Device lock',
    //         description: 'Settings related to screen lock and passwords.',
    //         learnMoreLink: '/tables/screenlock#apple',
    //         payloadOptions: [
    //           {
    //             name: 'Max inactivity time before device locks',
    //             uniqueSlug: 'windows-device-lock-max-inactivity',
    //             category: 'Device lock',
    //             tooltip: 'The number of seconds a device can remain inactive before a password is required to unlock the device.',
    //             supportedAccessTypes: ['add', 'replace'],
    //             acceptedValue: {
    //               type: 'number',
    //               maxValue: '9000',
    //               minValue: '1',
    //             }
    //           },
    //           {
    //             name: 'Require alphanumeric device password',
    //             uniqueSlug: 'windows-device-lock-require-alphanumeric-device-password',
    //             category: 'Device lock',
    //             supportedAccessTypes: ['add', 'replace'],
    //             acceptedValue: {
    //               type: 'radio',
    //               options: [
    //                 {
    //                   name: 'Password or alphanumeric PIN required',
    //                   value: '1'
    //                 },
    //                 {
    //                   name: 'Password or Numeric PIN required',
    //                   value: '2'
    //                 },
    //                 {
    //                   name: 'Password, Numeric PIN, or alphanumeric PIN required',
    //                   value: '3',
    //                 }
    //               ]
    //             }
    //           },
    //           {
    //             name: 'Enable device password',
    //             uniqueSlug: 'windows-device-lock-enable-device-password',
    //             tooltip: 'Require a password to unlock the device',
    //             category: 'Device lock',
    //             supportedAccessTypes: ['add', 'replace'],
    //             acceptedValue: {
    //               type: 'boolean',
    //             }
    //           }
    //         ]
    //       }
    //     ]
    //   },
    //   {
    //     name: 'Second category',
    //     subcategories: [
    //       {
    //         name: 'This is the same as the other one.',
    //         description: 'Settings related to screen lock and passwords.',
    //         learnMoreLink: '/tables/screenlock#apple',
    //         payloadOptions: [
    //           {
    //             name: 'Max inactivity time before device locks',
    //             uniqueSlug: 'windows-device-locking-max-inactivity-2',
    //             category: 'Device locking',
    //             tooltip: 'The number of seconds a device can remain inactive before a password is required to unlock the device.',
    //             supportedAccessTypes: ['add', 'replace'],
    //             acceptedValue: {
    //               type: 'number',
    //               maxValue: '9000',
    //               minValue: '1',
    //             }
    //           },
    //           {
    //             name: 'Require alphanumeric device password',
    //             uniqueSlug: 'windows-device-locking-require-alphanumeric-device-password-2',
    //             category: 'Device locking',
    //             supportedAccessTypes: ['add', 'replace'],
    //             acceptedValue: {
    //               type: 'radio',
    //               options: [
    //                 {
    //                   name: 'Password or alphanumeric PIN required',
    //                   value: '1'
    //                 },
    //                 {
    //                   name: 'Password or Numeric PIN required',
    //                   value: '2'
    //                 },
    //                 {
    //                   name: 'Password, Numeric PIN, or alphanumeric PIN required',
    //                   value: '3',
    //                 }
    //               ]
    //             }
    //           },
    //           {
    //             name: 'Enable device password',
    //             uniqueSlug: 'windows-device-locking-enable-device-password-2',
    //             tooltip: 'Require a password to unlock the device',
    //             category: 'Device locking',
    //             supportedAccessTypes: ['add', 'replace'],
    //             acceptedValue: {
    //               type: 'boolean',
    //             }
    //           }
    //         ]
    //       }
    //     ]
    //   },
    // ],

    // For UI demo with three options
    // apple payloads (empty array)
    applePayloads: [],

    // The current selected payload category, controls which options are shown in the middle section
    selectedPayloadCategory: undefined,

    // A list of the payloads that the user selected.
    // Used to build the inputs for the profile builder form.
    selectedPayloads: [],

    // A list of the payloads that are required to enforce a user selected paylaod.
    // Used to build the inputs for the profile builder form.
    autoSelectedPayloads: [],

    // Used to keep payloads grouped by category in the profile builder.
    selectedPayloadsGroupedByCategory: {},

    // Used to keep track of which payloads have been added to the profile builder. (Essentially formData for the payload selector)
    selectedPayloadSettings: {},

    // Used to keep track of which payloads have been automatically added to the profile builder
    autoSelectedPayloadSettings: {},

    // For the profile builder
    configurationBuilderFormData: {},
    configurationBuilderFormRules: {},
    // For the downlaod modal
    downloadProfileFormRules: {
      name: {required: true},
    },
    downloadProfileFormData: {},
    profileFilename: undefined,
    profileDescription: undefined,
    // windows payloads
    windowsPayloads: [
      {
        name: 'Enable device password',
        uniqueSlug: 'windows-device-lock-enable-device-lock',
        tooltip: 'Require a password to unlock the device',
        category: 'Device lock',
        supportedAccessTypes: ['add', 'replace'],
        formInput: {
          type: 'boolean',
          trueValue: 0,
          falseValue: 1
        },
        formOutput: {// For the compiler
          settingFormat: 'int',// Used to generate a configuration profile
          settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordEnabled',// Used to generate a configuration profile
          trueValue: 0,// (type=boolean only) Used to keep track of what values the boolean input represents.
          falseValue: 1,// (type=boolean only) Used to keep track of what values the boolean input represents.
        },
      },
      {
        name: 'Max inactivity time before device locks',
        uniqueSlug: 'windows-device-lock-max-inactivity-before-device-locks',
        category: 'Device lock',
        tooltip: 'The number of seconds a device can remain inactive before a password is required to unlock the device.',
        supportedAccessTypes: ['add', 'replace'],
        alsoAutoSetWhenSelected: [
          {
            dependingOnSettingSlug: 'windows-device-lock-enable-device-lock',
            dependingOnSettingValue: true,
          }
        ],
        formInput: {
          type: 'number',
          maxValue: '9000',
          minValue: '1',
        },
        formOutput: {// For the compiler
          settingFormat: 'int',// Used to generate a configuration profile
          settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordEnabled',// Used to generate a configuration profile
        },
      },
      {
        name: 'Require alphanumeric device password',
        uniqueSlug: 'windows-device-lock-require-alphanumeric-device-password',
        category: 'Device lock',
        supportedAccessTypes: ['add', 'replace'],
        alsoAutoSetWhenSelected: [
          {
            dependingOnSettingSlug: 'windows-device-lock-enable-device-lock',
            dependingOnSettingValue: true,
          }
        ],
        formInput: {
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
        },
        formOutput: {// For the compiler
          settingFormat: 'int',// Used to generate a configuration profile
          settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/AlphanumericDevicePasswordRequired',// Used to generate a configuration profile
        },
      },
      {
        name: 'Min password length',
        toolTip: 'The minimum number of characters a device\'s password must be',
        supportedAccessTypes: ['ADD', 'REPLACE'],
        alsoAutoSetWhenSelected: [
          {
            dependingOnSettingSlug: 'windows-device-lock-enable-device-lock',
            dependingOnSettingValue: true,
          }
        ],
        formInput: {
          type: 'number',
          defaultValue: 4,
          minValue: 4,
          maxValue: 16
        },
        formOutput: {// For the compiler
          settingFormat: 'int',// Used to generate a configuration profile
          settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength',// Used to generate a configuration profile
        },
      }
    ]
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
      if(this.selectedPayloadOptions[payloadOption.uniqueSlug]) {
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
    _getWindowsXmlPayloadString: function(payload) {
      let windowsPayloadTemplate = `
<${_.capitalize(payload.formData.accessType)}>
  <!-- ${payload.name} -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">${payload.formOutput.settingFormat}</Format>
    </Meta>
    <Target>
      <LocURI>${payload.formOutput.settingTarget}</LocURI>
    </Target>
    <Data>${payload.formData.value}</Data>
  </Item>
</${_.capitalize(payload.formData.accessType)}>
`;
      return _.trim(windowsPayloadTemplate);
    },
    handleSubmittingDownloadProfileForm: async function() {
      this.syncing = true;
      if(this.selectedPlatform === 'windows') {
        await this.buildWindowsProfile();
      }
    },
    buildWindowsProfile: function() {
      let xmlString = '';
      // Iterate through the selcted payloads
      for(let payload of this.selectedPayloads) {
        let payloadToAdd = _.clone(payload);
        // Get the selected access type for this payload
        let accessType = this.configurationBuilderFormData[payload.uniqueSlug+'-access-type'];
        // Get the selected value for this payload
        let value = this.configurationBuilderFormData[payload.uniqueSlug+'-value'];
        // If this payload is a boolean input, we'll convert the true/false value into the expected value for this payload.
        if(payload.formInput.type === 'boolean'){
          if(value) {
            value = payload.formOutput.trueValue;
          } else {
            value = payload.formOutput.falseValue;
          }
        }
        payloadToAdd.formData = {accessType, value};
        let outputForThisPayload = this._getWindowsXmlPayloadString(payloadToAdd);
        xmlString += outputForThisPayload + '\n';
      }
      let xmlDownloadUrl = URL.createObjectURL(new Blob([_.trim(xmlString)], { type: 'text/xml;' }));
      let exportDownloadLink = document.createElement('a');
      exportDownloadLink.href = xmlDownloadUrl;
      exportDownloadLink.download = `${this.downloadProfileFormData.name}.xml`;
      exportDownloadLink.click();
      URL.revokeObjectURL(xmlDownloadUrl);
      this.syncing = false;
    },
    clickRemoveAllPayloadOptions: function() {
      this.selectedPayloadOptions = {};
      this.payloadOptionsToDisplay = [];
      this.payloadOptionsToDisplayGroupedByCategory = {};
    },
    clickRemoveOneCategoryPayloadOptions: function(category) {
      let optionsToRemove = this.payloadOptionsToDisplayGroupedByCategory[category];
      this.payloadOptionsToDisplayGroupedByCategory = _.without(this.payloadOptionsToDisplayGroupedByCategory, category);
      for(let option of optionsToRemove){
        this.payloadOptionsToDisplay = _.without(this.payloadOptionsToDisplay, option);
        this.selectedPayloadOptions[option.uniqueSlug] = false;
      }
      this.payloadOptionsToDisplayGroupedByCategory = _.groupBy(this.payloadOptionsToDisplay, 'category');
    },
    clickRemovePayloadOption: function(option) {
      let payloadToRemove = _.find(this.selectedPayloads, {uniqueSlug: option.uniqueSlug});
      // check the alsoAutoSetWhenSelected value of the payload we're removing.
      let newSelectedPayloads = _.without(this.selectedPayloads, payloadToRemove);
      this.selectedPayloadSettings[option.uniqueSlug] = false;
      this.selectedPayloads = _.uniq(newSelectedPayloads);
      this.selectedPayloadsGroupedByCategory = _.groupBy(this.selectedPayloads, 'category');
      delete this.configurationBuilderFormRules[option.uniqueSlug+'-value'];
      delete this.configurationBuilderFormRules[option.uniqueSlug+'-access-type'];
    },
    handleSubmittingConfigurationBuilderForm: function() {
      if(_.keysIn(this.selectedPayloadsGroupedByCategory).length > 1) {
        this.modal = 'multiple-payloads-selected';
      } else {
        this.modal = 'download-profile';
      }
    },
    clickClearOneFormError: async function(field) {
      await this.forceRender();
      if(this.formErrors[field]){
        this.formErrors = _.omit(this.formErrors, field);
      }
    },
    clickSelectPayloadCategory: function(payloadGroup) {
      this.selectedPayloadCategory = payloadGroup;
    },
    clickSelectPayload: async function(payloadSlug) {
      if(!this.selectedPayloadSettings[payloadSlug]){
        let payloadsToUse;
        if(this.selectedPlatform === 'windows'){
          payloadsToUse = this.windowsPayloads;
        } else {
          payloadsToUse = this.applePayloads;
        }
        let selectedPayload = _.find(payloadsToUse, {uniqueSlug: payloadSlug}) || {};
        if(selectedPayload.alsoAutoSetWhenSelected) {
          for(let autoSelectedPayload of selectedPayload.alsoAutoSetWhenSelected ) {
            let payloadToAddSlug = autoSelectedPayload.dependingOnSettingSlug;
            let payloadToAdd = _.find(payloadsToUse, {uniqueSlug: payloadToAddSlug});
            this.selectedPayloads.push(payloadToAdd);
            this.$set(this.configurationBuilderFormData, payloadToAddSlug+'-value', autoSelectedPayload.dependingOnSettingValue);
            this.autoSelectedPayloadSettings[payloadToAddSlug] = true;
            this.selectedPayloadSettings[payloadToAddSlug] = true;
            this.configurationBuilderFormRules[payloadToAddSlug+'-value'] = {required: true};
            this.configurationBuilderFormRules[payloadToAddSlug+'-access-type'] = {required: true};
          }
        }
        this.selectedPayloads.push(selectedPayload);
        this.selectedPayloads = _.uniq(this.selectedPayloads);
        this.configurationBuilderFormRules[selectedPayload.uniqueSlug+'-value'] = {required: true};
        this.configurationBuilderFormRules[selectedPayload.uniqueSlug+'-access-type'] = {required: true};
        this.selectedPayloadsGroupedByCategory = _.groupBy(this.selectedPayloads, 'category');
        this.selectedPayloadSettings[payloadSlug] = true;
        // console.log(this.configurationBuilderFormData);
      } else {

        // Remove the payload option and all dependencies
        let payloadToRemove = _.find(this.selectedPayloads, {uniqueSlug: payloadSlug});
        console.log(payloadSlug, payloadToRemove);
        // check the alsoAutoSetWhenSelected value of the payload we're removing.
        let newSelectedPayloads = _.without(this.selectedPayloads, payloadToRemove);
        delete this.configurationBuilderFormRules[payloadSlug+'-value'];
        delete this.configurationBuilderFormRules[payloadSlug+'-access-type'];

        // if (_.difference(_.uniq(_.pluck(this.selectedPayloads, 'alsoAutoSetWhenSelected')), _.uniq(_.pluck(this.newSelectedPayloads, 'alsoAutoSetWhenSelected'))).length === 0){
        //   // dependencies are the same.
        // } else {
        //   // TODO: also handle cases where an auto selected payload is manually selected
        //   // console.log(_.difference(_.uniq(_.pluck(this.selectedPayloads, 'alsoAutoSetWhenSelected')), _.uniq(_.pluck(this.newSelectedPayloads, 'alsoAutoSetWhenSelected'))));
        //   let removedAutoSetSettings = _.difference(_.uniq(_.pluck(this.selectedPayloads, 'alsoAutoSetWhenSelected')), _.uniq(_.pluck(this.newSelectedPayloads, 'alsoAutoSetWhenSelected')));
        //   console.log(removedAutoSetSettings);
        //   // // Dependencies are different
        //   for(let removedSetting in removedAutoSetSettings){
        //     console.log('rms', removedSetting);
        //     if(removedSetting){
        //       let newPayloadToRemove = _.find(this.selectedPayloads, {uniqueSlug: removedSetting.dependingOnSettingSlug});
        //       delete this.configurationBuilderFormRules[removedSetting.dependingOnSettingSlug+'-value'];
        //       delete this.configurationBuilderFormRules[removedSetting.dependingOnSettingSlug+'-access-type'];
        //       this.selectedPayloadSettings[removedSetting.dependingOnSettingSlug] = false;
        //       this.autoSelectedPayloadSettings[removedSetting.dependingOnSettingSlug] = false;
        //       newSelectedPayloads = _.without(this.selectedPayloads, newPayloadToRemove);
        //     }
        //   }
        // }
        this.selectedPayloadSettings[payloadSlug] = false;
        this.selectedPayloads = _.uniq(newSelectedPayloads);
        this.selectedPayloadsGroupedByCategory = _.groupBy(this.selectedPayloads, 'category');
      }
      await this.forceRender();
    },
    clickOpenResetFormModal: function() {
      this.modal = 'reset-form';
    },
    clickResetForm: async function() {
      this.step = 'platform-select';
      this.platform = undefined;
      // The current selected payload category, controls which options are shown in the middle section
      this.selectedPayloadCategory = undefined;

      // A list of the payloads that the user selected.
      // Used to build the inputs for the profile builder form.
      this.selectedPayloads = [];

      // A list of the payloads that are required to enforce a user selected paylaod.
      // Used to build the inputs for the profile builder form.
      this.autoSelectedPayloads = [];

      // Used to keep payloads grouped by category in the profile builder.
      this.selectedPayloadsGroupedByCategory = {};

      // Used to keep track of which payloads have been added to the profile builder. (Essentially formData for the payload selector)
      this.selectedPayloadSettings = {};

      // Used to keep track of which payloads have been automatically added to the profile builder
      this.autoSelectedPayloadSettings = {};

      // For the profile builder
      this.configurationBuilderFormData = {};
      this.configurationBuilderFormRules = {};
      this.modal = undefined;
      await this.forceRender();
    }
  }
});

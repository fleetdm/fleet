parasails.registerPage('configuration-builder', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // For the platform selector step.
    selectedPlatform: undefined,
    // selectedPlatform: 'windows',
    step: 'platform-select',
    // step: 'configuration-builder',
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

    // For modals
    modal: undefined,
    // For QAing modals
    // modal: 'download-profile',
    // modal: 'multiple-payloads-selected',



    // The current selected payload category, controls which options are shown in the middle section
    selectedPayloadCategory: undefined,

    // The current expanded list of subcategories
    expandedCategory: undefined,

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
    // For the download modal
    downloadProfileFormRules: {
      name: {required: true},
    },
    downloadProfileFormData: {},
    profileFilename: undefined,
    profileDescription: undefined,
    // mac OS payloads.
    macOSPayloads: [
      {
        name: 'Require device password',
        uniqueSlug: 'macos-enable-force-pin',
        tooltip: 'Require a password to unlock the device',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'boolean',
          trueValue: 0,
          falseValue: 1
        },
        formOutput: {// For the compiler
          settingFormat: 'boolean',// Used to generate a configuration profile
          settingKey: 'forcePIN',// Used to generate a configuration profile
          trueValue: '<true/>',// (type=boolean only) Used to keep track of what values the boolean input represents.
          falseValue: '<false/>',// (type=boolean only) Used to keep track of what values the boolean input represents.
        },
      },
      {
        name: 'Allow simple password',
        uniqueSlug: 'macos-enable-allow-simple-pin',
        tooltip: 'If false, the system prevents use of a simple passcode. A simple passcode contains repeated characters, or increasing or decreasing characters, such as 123 or CBA.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'boolean',
          trueValue: 0,
          falseValue: 1
        },
        formOutput: {// For the compiler
          settingFormat: 'boolean',// Used to generate a configuration profile
          settingKey: 'allowSimple',// Used to generate a configuration profile
          trueValue: '<true/>',// (type=boolean only) Used to keep track of what values the boolean input represents.
          falseValue: '<false/>',// (type=boolean only) Used to keep track of what values the boolean input represents.
        },
      },
      {
        name: 'Max inactivity time before device locks',
        uniqueSlug: 'macos-max-inactivity',
        tooltip: 'The maximum number of minutes for which the device can be idle without the user unlocking it, before the system locks it.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'number',
          defaultValue: 4,
          minValue: 0,
          maxValue: 60,
          unitLabel: 'minutes'
        },
        formOutput: {// For the compiler
          settingFormat: 'integer',// Used to generate a configuration profile
          settingKey: 'maxInactivity',// Used to generate a configuration profile
        },
      },
      {
        name: 'Minimum password length',
        uniqueSlug: 'macos-min-length',
        tooltip: 'The minimum overall length of the passcode.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'number',
          defaultValue: 0,
          minValue: 0,
          maxValue: 16,
          unitLabel: 'characters'
        },
        formOutput: {// For the compiler
          settingFormat: 'integer',// Used to generate a configuration profile
          settingKey: 'minLength',// Used to generate a configuration profile
        },
      },
      {
        name: 'Require alphanumeric password',
        uniqueSlug: 'macos-require-alphanumeric-password',
        tooltip: 'If true, the system requires alphabetic characters instead of only numeric characters.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'boolean',
          trueValue: 0,
          falseValue: 1
        },
        formOutput: {// For the compiler
          settingFormat: 'boolean',// Used to generate a configuration profile
          settingKey: 'requireAlphanumeric',// Used to generate a configuration profile
          trueValue: '<true/>',// (type=boolean only) Used to keep track of what values the boolean input represents.
          falseValue: '<false/>',// (type=boolean only) Used to keep track of what values the boolean input represents.
        },
      },
      {
        name: 'Change passcode at next login',
        uniqueSlug: 'macos-change-at-next-auth',
        tooltip: 'If true, the system causes a password reset to occur the next time the user tries to authenticate.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'boolean',
          trueValue: 0,
          falseValue: 1
        },
        formOutput: {// For the compiler
          settingFormat: 'boolean',// Used to generate a configuration profile
          settingKey: 'changeAtNextAuth',// Used to generate a configuration profile
          trueValue: '<true/>',// (type=boolean only) Used to keep track of what values the boolean input represents.
          falseValue: '<false/>',// (type=boolean only) Used to keep track of what values the boolean input represents.
        },
      },
      {
        name: 'Maximum number of failed attempts',
        uniqueSlug: 'macos-max-failed-attempts',
        tooltip: 'The number of allowed failed attempts to enter the passcode at the device’s lock screen. After four failed attempts, the system imposes a time delay before a passcode can be entered again. When this number is exceeded in macOS, the system locks the device.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'number',
          defaultValue: 11,
          minValue: 2,
          maxValue: 11,
          unitLabel: 'attempts'
        },
        formOutput: {// For the compiler
          settingFormat: 'integer',// Used to generate a configuration profile
          settingKey: 'maxFailedAttempts',// Used to generate a configuration profile
        },
      },
      {
        name: 'Max grace period',
        uniqueSlug: 'macos-max-grace-period',
        tooltip: 'The maximum grace period, in minutes, to unlock the device without entering a passcode. The default is 0, which is no grace period and requires a passcode immediately.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'number',
          defaultValue: 0,
          minValue: 0,
          maxValue: 999,
          unitLabel: 'minutes'
        },
        formOutput: {// For the compiler
          settingFormat: 'integer',// Used to generate a configuration profile
          settingKey: 'maxGracePeriod',// Used to generate a configuration profile
        },
      },
      {
        name: 'Max passcode age',
        uniqueSlug: 'macos-max-pin-age',
        tooltip: 'The number of days for which the passcode can remain unchanged. After this number of days, the system forces the user to change the passcode before it unlocks the device.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'number',
          defaultValue: 0,
          minValue: 0,
          maxValue: 999,
          unitLabel: 'days'
        },
        formOutput: {// For the compiler
          settingFormat: 'integer',// Used to generate a configuration profile
          settingKey: 'maxPINAgeInDays',// Used to generate a configuration profile
        },
      },
      {
        name: 'Minimum complex characters',
        uniqueSlug: 'macos-min-complex-characters',
        tooltip: 'The minimum number of complex characters that a passcode needs to contain. A complex character is a character other than a number or a letter, such as &, %, $, and #.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'number',
          defaultValue: 0,
          minValue: 0,
          maxValue: 4,
          unitLabel: 'characters'
        },
        formOutput: {// For the compiler
          settingFormat: 'integer',// Used to generate a configuration profile
          settingKey: 'minComplexChars',// Used to generate a configuration profile
        },
      },
      {
        name: 'Minutes until failed login reset',
        uniqueSlug: 'macos-minutes-until-failed-login-reset',
        tooltip: 'The number of minutes before the system resets the login after the maximum number of unsuccessful login attempts is reached.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'number',
          defaultValue: 0,
          minValue: 0,
          maxValue: 4,
          unitLabel: 'minutes'
        },
        formOutput: {// For the compiler
          settingFormat: 'integer',// Used to generate a configuration profile
          settingKey: 'minutesUntilFailedLoginReset',// Used to generate a configuration profile
        },
      },
      {
        name: 'Passcode history',
        uniqueSlug: 'macos-passcode-history',
        tooltip: 'This value defines N, where the new passcode must be unique within the last N entries in the passcode history.',
        category: 'Device lock',
        payload: 'Passcode',
        payloadType: 'com.apple.mobiledevice.passwordpolicy',
        formInput: {
          type: 'number',
          minValue: 1,
          maxValue: 50,
        },
        formOutput: {// For the compiler
          settingFormat: 'integer',// Used to generate a configuration profile
          settingKey: 'pinHistory',// Used to generate a configuration profile
        },
      },
    ],
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
          maxValue: 9000,
          minValue: 1,
          unitLabel: 'seconds'
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
        uniqueSlug: 'windows-device-lock-min-password-length',
        category: 'Device lock',
        supportedAccessTypes: ['add', 'replace'],
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
          maxValue: 16,
          unitLabel: 'characters'
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
    handleSubmittingDownloadProfileForm: async function() {
      this.syncing = true;
      if(this.selectedPlatform === 'windows') {
        await this.buildWindowsProfile();
      } else if(this.selectedPlatform === 'macos') {
        await this.buildMacOSProfile();
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
    buildMacOSProfile: function() {
      let xmlString = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
<key>PayloadContent</key>
<array>
`;
      let uuidForThisProfile = crypto.randomUUID();
      // Iterate through the selcted payloads
      // group selected payloads by their payload type value.
      let payloadsToCreateDictonariesFor = _.groupBy(this.selectedPayloads, 'payloadType');
      for(let optionsInTheSamePayload in payloadsToCreateDictonariesFor) {
        // First build the payloadDisplayName, payloadIdentifier, payloadType, payloadUUID, and payloadVersion keys.

        let uuidForThisPayload = crypto.randomUUID();
        let dictionaryStringForThisPayload = `<dict>
<key>PayloadDisplayName</key>
<string>${payloadsToCreateDictonariesFor[optionsInTheSamePayload][0].payload}</string>
<key>PayloadIdentifier</key>
<string>${payloadsToCreateDictonariesFor[optionsInTheSamePayload][0].payloadType + '.' + uuidForThisPayload}</string>
<key>PayloadType</key>
<string>${payloadsToCreateDictonariesFor[optionsInTheSamePayload][0].payloadType}</string>
<key>PayloadUUID</key>
<string>${uuidForThisPayload}</string>
<key>PayloadVersion</key>
<integer>1</integer>
`;
        for(let payloadOption of payloadsToCreateDictonariesFor[optionsInTheSamePayload]) {
          let payloadToAdd = _.clone(payloadOption);
          let value = this.configurationBuilderFormData[payloadOption.uniqueSlug+'-value'];
          if(payloadOption.formInput.type === 'boolean') {
            if(value) {
              value = payloadOption.formOutput.trueValue;
            } else {
              value = payloadOption.formOutput.falseValue;
            }
          }
          dictionaryStringForThisPayload += `<key>${payloadToAdd.formOutput.settingKey}</key>
`;
          if(payloadToAdd.formOutput.settingFormat === 'boolean'){
            dictionaryStringForThisPayload += `${value}
`;
          } else {
            dictionaryStringForThisPayload += `<${payloadToAdd.formOutput.settingFormat}>${value}</${payloadToAdd.formOutput.settingFormat}>
`;
          }
        }
        dictionaryStringForThisPayload += `</dict>
`;
        // If this payload is a boolean input, we'll convert the true/false value into the expected value for this payload.
        xmlString += dictionaryStringForThisPayload;
      }
      xmlString += `</array>
<key>PayloadDisplayName</key>
<string>${this.downloadProfileFormData.name}</string>
<key>PayloadDescription</key>
<string>${this.downloadProfileFormData.description}</string>
<key>PayloadIdentifier</key>
<string>Fleet-profile-generator.${uuidForThisProfile}</string>
<key>PayloadType</key>
<string>Configuration</string>
<key>PayloadUUID</key>
<string>${uuidForThisProfile}</string>
<key>PayloadVersion</key>
<integer>1</integer>
<key>TargetDeviceType</key>
<integer>5</integer>
</dict>
</plist>`;
      let xmlDownloadUrl = URL.createObjectURL(new Blob([_.trim(xmlString)], { type: 'text/xml;' }));
      let exportDownloadLink = document.createElement('a');
      exportDownloadLink.href = xmlDownloadUrl;
      exportDownloadLink.download = `${this.downloadProfileFormData.name}.mobileconfig`;
      exportDownloadLink.click();
      URL.revokeObjectURL(xmlDownloadUrl);
      this.syncing = false;
    },
    clickRemoveOneCategoryPayloadOptions: function(category) {
      let optionsToRemove = this.selectedPayloadsGroupedByCategory[category];
      this.selectedPayloadsGroupedByCategory = _.without(this.selectedPayloadsGroupedByCategory, category);
      for(let option of optionsToRemove){
        let newSelectedPayloads = _.without(this.selectedPayloads, option);
        this.selectedPayloadSettings[option.uniqueSlug] = false;
        this.selectedPayloads = _.uniq(newSelectedPayloads);
        delete this.configurationBuilderFormRules[option.uniqueSlug+'-value'];
        delete this.configurationBuilderFormData[option.uniqueSlug+'-value'];
        if(this.selectedPlatform === 'windows') {
          delete this.configurationBuilderFormRules[option.uniqueSlug+'-access-type'];
          delete this.configurationBuilderFormData[option.uniqueSlug+'-access-type'];
        }
      }
      this.selectedPayloadsGroupedByCategory = _.groupBy(this.payloadOptionsToDisplay, 'category');
      console.log(this.selectedPayloadsGroupedByCategory);
    },
    clickRemovePayloadOption: function(option) {
      let payloadToRemove = _.find(this.selectedPayloads, {uniqueSlug: option.uniqueSlug});
      // check the alsoAutoSetWhenSelected value of the payload we're removing.
      let newSelectedPayloads = _.without(this.selectedPayloads, payloadToRemove);
      this.selectedPayloadSettings[option.uniqueSlug] = false;
      this.selectedPayloads = _.uniq(newSelectedPayloads);
      this.selectedPayloadsGroupedByCategory = _.groupBy(this.selectedPayloads, 'category');
      delete this.configurationBuilderFormRules[option.uniqueSlug+'-value'];
      if(this.selectedPlatform === 'windows') {
        delete this.configurationBuilderFormRules[option.uniqueSlug+'-access-type'];
      }
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
      this._enablePayloadToolTips();
    },
    _enablePayloadToolTips: async function() {
      await setTimeout(()=>{
        $('[data-toggle="tooltip"]').tooltip({
          container: '#configuration-builder',
          trigger: 'hover',
        });
      }, 400);
    },
    clickSelectPayload: async function(payloadSlug) {
      if(!this.selectedPayloadSettings[payloadSlug]){
        let payloadsToUse;
        if(this.selectedPlatform === 'windows'){
          payloadsToUse = this.windowsPayloads;
        } else if(this.selectedPlatform === 'macos') {
          payloadsToUse = this.macOSPayloads;
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
            if(this.selectedPlatform === 'windows') {
              this.configurationBuilderFormRules[payloadToAddSlug+'-access-type'] = {required: true};
            }
          }
        }
        this.selectedPayloads.push(selectedPayload);
        this.selectedPayloads = _.uniq(this.selectedPayloads);
        this.configurationBuilderFormRules[selectedPayload.uniqueSlug+'-value'] = {required: true};
        if(selectedPayload.formInput.type === 'boolean'){
          // default boolean inputs to false.
          this.configurationBuilderFormData[selectedPayload.uniqueSlug+'-value'] = false;
        } else if(selectedPayload.formInput.type === 'number') {
          this.configurationBuilderFormData[selectedPayload.uniqueSlug+'-value'] = selectedPayload.formInput.defaultValue;
        }
        if(this.selectedPlatform === 'windows') {
          this.configurationBuilderFormRules[selectedPayload.uniqueSlug+'-access-type'] = {required: true};
        }
        this.selectedPayloadsGroupedByCategory = _.groupBy(this.selectedPayloads, 'category');
        this.selectedPayloadSettings[payloadSlug] = true;
        // console.log(this.configurationBuilderFormData);
      } else {
        // Remove the payload option and all dependencies
        let payloadToRemove = _.find(this.selectedPayloads, {uniqueSlug: payloadSlug});
        // check the alsoAutoSetWhenSelected value of the payload we're removing.
        let newSelectedPayloads = _.without(this.selectedPayloads, payloadToRemove);
        delete this.configurationBuilderFormRules[payloadSlug+'-value'];
        if(this.selectedPlatform === 'windows') {
          delete this.configurationBuilderFormRules[payloadSlug+'-access-type'];
        }
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
      this.platformSelectFormData.platform = undefined;
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
    },
    closeModal: function() {
      this.modal = undefined;
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
  }
});

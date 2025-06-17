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
    macosCategoriesAndPayloads: [
      {
        categoryName: 'Privacy & security',
        categorySlug: 'macos-privacy-and-security',
        subcategories: [
          {
            subcategoryName: 'Device lock',
            subcategorySlug: 'macos-device-lock',
            description: 'Settings related to screen lock and passwords.',
            learnMoreLinkUrl: 'https://developer.apple.com/documentation/devicemanagement/passcode',
            payloads: [
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
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'forcePIN',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
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
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'allowSimple',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
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
                formOutput: {
                  settingFormat: 'integer',
                  settingKey: 'maxInactivity',
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
                formOutput: {
                  settingFormat: 'integer',
                  settingKey: 'minLength',
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
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'requireAlphanumeric',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
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
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'changeAtNextAuth',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
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
                formOutput: {
                  settingFormat: 'integer',
                  settingKey: 'maxFailedAttempts',
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
                formOutput: {
                  settingFormat: 'integer',
                  settingKey: 'maxGracePeriod',
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
                formOutput: {
                  settingFormat: 'integer',
                  settingKey: 'maxPINAgeInDays',
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
                formOutput: {
                  settingFormat: 'integer',
                  settingKey: 'minComplexChars',
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
                formOutput: {
                  settingFormat: 'integer',
                  settingKey: 'minutesUntilFailedLoginReset',
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
                formOutput: {
                  settingFormat: 'integer',
                  settingKey: 'pinHistory',
                },
              },
            ],
          }
        ]
      },
    ],
    // windows payloads
    windowsCategoriesAndPayloads: [
      {
        categoryName: 'Privacy & security',
        categorySlug: 'windows-privacy-and-security',
        subcategories: [
          {
            subcategoryName: 'Device lock',
            subcategorySlug: 'windows-device-lock',
            description: 'Settings related to screen lock and passwords.',
            learnMoreLinkUrl: 'https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-devicelock',
            payloads: [
              {
                name: 'Enable device password',
                uniqueSlug: 'windows-device-lock-enable-device-lock',
                tooltip: 'Require a password to unlock the device',
                category: 'Device lock',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordEnabled',
                  trueValue: 0,
                  falseValue: 1,
                },
              },
              {
                name: 'Device password expiration',
                uniqueSlug: 'windows-device-lock-device-password-expiration',
                tooltip: 'Specifies when the password expires (in days).',
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
                  maxValue: 730,
                  minValue: 1,
                  unitLabel: 'days'
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordExpiration',
                },
              },
              {
                name: 'Device password history',
                uniqueSlug: 'windows-device-lock-device-password-history',
                tooltip: `Specifies how many passwords can be stored in the history that can't be used. \n The value includes the user's current password. This value denotes that with a setting of 1, the user can't reuse their current password when choosing a new password, while a setting of 5 means that a user can't set their new password to their current password or any of their previous four passwords.`,
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
                  maxValue: 50,
                  minValue: 0,
                  unitLabel: 'passwords'
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordHistory',
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
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock',
                },
              },
              {
                name: 'Max inactivity time before device locks with external display',
                uniqueSlug: 'windows-device-lock-max-inactivity-before-device-locks-with-external-display',
                category: 'Device lock',
                tooltip: 'The number of seconds a device can remain inactive while using an external monitor before a password is required to unlock the device.',
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
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLockWithExternalDisplay',
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
                      value: 0
                    },
                    {
                      name: 'Password or Numeric PIN required',
                      value: 1
                    },
                    {
                      name: 'Password, Numeric PIN, or alphanumeric PIN required',
                      value: 2,
                    }
                  ]
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/AlphanumericDevicePasswordRequired',
                },
              },
              {
                name: 'Max failed attempts',
                toolTip: 'The number of authentication failures allowed before the device will be wiped. A value of 0 disables device wipe functionality.',
                uniqueSlug: 'windows-device-lock-max-failed-attempts',
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
                  defaultValue: 0,
                  minValue: 0,
                  maxValue: 999,
                  unitLabel: 'attempts'
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxDevicePasswordFailedAttempts',
                },
              },
              {
                name: 'Max password age',
                toolTip: `Determines the period of time (in days) that a password can be used before the system requires the user to change it. You can set passwords to expire after a number of days between 1 and 999, or you can specify that passwords never expire by setting the number of days to 0.`,
                uniqueSlug: 'windows-device-lock-max-password-age',
                category: 'Device lock',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'number',
                  defaultValue: 42,
                  minValue: 0,
                  maxValue: 999,
                  unitLabel: 'days'
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/MaximumPasswordAge',
                },
              },
              {
                name: 'Min password age',
                toolTip: `Determines the period of time (in days) that a password must be used before the user can change it. You can set a value between 1 and 998 days, or you can allow changes immediately by setting the number of days to 0. If the maximum password age is set to 0, the minimum password age can be set to any value between 0 and 998. Configure the minimum password age to be more than 0 if you want Enforce password history to be effective.`,
                uniqueSlug: 'windows-device-lock-min-password-age',
                category: 'Device lock',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'number',
                  defaultValue: 1,
                  minValue: 0,
                  maxValue: 998,
                  unitLabel: 'days'
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/MinimumPasswordAge',
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
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength',
                },
              },
              {
                name: 'Min number of types of complex characters in device password',
                toolTip: `The number of complex element types (uppercase and lowercase letters, numbers, and punctuation) required for a strong PIN or password.`,
                uniqueSlug: 'windows-device-min-types-of-complex-characters',
                category: 'Device lock',
                supportedAccessTypes: ['add', 'replace'],
                alsoAutoSetWhenSelected: [
                  {
                    dependingOnSettingSlug: 'windows-device-lock-enable-device-lock',
                    dependingOnSettingValue: true,
                  },
                  {
                    dependingOnSettingSlug: 'windows-device-lock-require-alphanumeric-device-password',
                    dependingOnSettingValue: 0,
                  }
                ],
                formInput: {
                  type: 'radio',
                  options: [
                    {
                      name: 'Digits only',
                      value: 1
                    },
                    {
                      name: 'Digits and lowercase letters are required.',
                      value: 2
                    },
                    {
                      name: 'Digits lowercase letters and uppercase letters are required.',
                      value: 3,
                    },
                    {
                      name: 'Digits lowercase letters uppercase letters and special characters are required. Not supported in desktop.',
                      value: 4,
                    }
                  ]
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordComplexCharacters',
                },
              },
              {
                name: 'Allow simple device password',
                toolTip: `Specifies whether PINs or passwords such as 1111 or 1234 are allowed. For the desktop, it also controls the use of picture passwords.`,
                uniqueSlug: 'windows-device-lock-allow-simple-device-password',
                category: 'Device lock',
                supportedAccessTypes: ['add', 'replace'],
                alsoAutoSetWhenSelected: [
                  {
                    dependingOnSettingSlug: 'windows-device-lock-enable-device-lock',
                    dependingOnSettingValue: true,
                  }
                ],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/AllowSimpleDevicePassword',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Clear text password',
                toolTip: `This security setting determines whether the operating system stores passwords using reversible encryption. Storing passwords using reversible encryption is essentially the same as storing plaintext versions of the passwords. For this reason, this policy should never be enabled unless application requirements outweigh the need to protect password information.`,
                uniqueSlug: 'windows-device-lock-clear-text-password',
                category: 'Device lock',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/AllowSimpleDevicePassword',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Prevent enabling lock screen camera',
                toolTip: `Disables the lock screen camera toggle switch in PC Settings and prevents a camera from being invoked on the lock screen.`,
                uniqueSlug: 'windows-device-lock-disable-screen-camera',
                category: 'Device lock',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/AllowSimpleDevicePassword',
                  trueValue: '<![CDATA[<enabled/>]]>',
                  falseValue: '<![CDATA[<disabled/>]]>',
                },
              },
              {
                name: 'Password must meet complexity requirements',
                toolTip: `If this policy is enabled, passwords must meet the following minimum requirements:
                    - Not contain the user's account name or parts of the user's full name that exceed two consecutive characters
                    - Be at least six characters in length
                    - Contain characters from three of the following four categories:
                      - English uppercase characters (A through Z)
                      - English lowercase characters (a through z)
                      - Base 10 digits (0 through 9)
                      - Non-alphabetic characters (for example, !, $, #, %)`,
                uniqueSlug: 'windows-device-lock-password-complexity',
                category: 'Device lock',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/DeviceLock/PasswordComplexity',
                  trueValue: 0,
                  falseValue: 1,
                },
              },
            ],
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
<string>${this.downloadProfileFormData.identifier}.${this.downloadProfileFormData.uuid}</string>
<key>PayloadType</key>
<string>Configuration</string>
<key>PayloadUUID</key>
<string>${this.downloadProfileFormData.uuid}</string>
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
        this.openDownloadModal();
      }
    },
    openDownloadModal: function() {
      this.modal = 'download-profile';
      if(this.selectedPlatform === 'macos'){
        this.downloadProfileFormRules = {
          name: {required: true},
          uuid: {required: true},
          identifier: {required: true},
        };
        // Generate a uuid to prefill for the download profile form.
        this.downloadProfileFormData.uuid = crypto.randomUUID();
      }
      this._enableToolTips();
    },
    clickClearOneFormError: async function(field) {
      await this.forceRender();
      if(this.formErrors[field]){
        this.formErrors = _.omit(this.formErrors, field);
      }
    },
    clickSelectPayloadCategory: function(payloadGroup) {
      this.selectedPayloadCategory = payloadGroup;
      this._enableToolTips();
    },
    _enableToolTips: async function() {
      await setTimeout(()=>{
        $('[data-toggle="tooltip"]').tooltip({
          container: '#configuration-builder',
          trigger: 'hover',
        });
      }, 400);
    },
    clickSelectPayload: async function(payloadSlug) {
      if(!this.selectedPayloadSettings[payloadSlug]){
        // if(this.selectedPlatform === 'windows'){
        //   payloadsToUse = this.windowsCategoriesAndPayloads;
        // } else if(this.selectedPlatform === 'macos') {
        //   payloadsToUse = this.macosCategoriesAndPayloads;
        // }
        let selectedPayload = _.find(this.selectedPayloadCategory.payloads, {uniqueSlug: payloadSlug}) || {};
        if(selectedPayload.alsoAutoSetWhenSelected) {
          for(let autoSelectedPayload of selectedPayload.alsoAutoSetWhenSelected ) {
            let payloadToAddSlug = autoSelectedPayload.dependingOnSettingSlug;
            let payloadToAdd = _.find(this.selectedPayloadCategory.payloads, {uniqueSlug: payloadToAddSlug});
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

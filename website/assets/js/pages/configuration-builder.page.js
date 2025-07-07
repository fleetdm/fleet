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

    currentSelectedCategoryForDownload: '',
    configurationBuilderFormDataByCategory: {},
    configurationBuilderByCategoryFormRules: {},
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
      {
        categoryName: 'Software & updates',
        categorySlug: 'macos-software-and-updates',
        subcategories: [
          {
            subcategoryName: 'Gatekeeper',
            subcategorySlug: 'macos-gatekeeper',
            description: 'Settings related to Gatekeeper',
            learnMoreLinkUrl: 'https://developer.apple.com/documentation/devicemanagement/systempolicycontrol',
            payloads: [
              {
                name: 'Enable Gatekeeper',
                uniqueSlug: 'macos-enable-gatekeeper',
                tooltip: 'If true, enables Gatekeeper. If false, disables Gatekeeper.',
                category: 'Gatekeeper',
                payload: 'Control',
                payloadType: 'com.apple.systempolicy.control',
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'EnableAssessment',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
                },
              },
              {
                name: 'Allow identified developers',
                uniqueSlug: 'macos-allow-identified-developers',
                tooltip: 'If true, enables Gatekeeper’s “Mac App Store and identified developers” option. \n If false, enables Gatekeeper’s “Mac App Store” option.',
                category: 'Gatekeeper',
                payload: 'Control',
                payloadType: 'com.apple.systempolicy.control',
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'AllowIdentifiedDevelopers',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
                },
              },
              {
                name: 'Enable XProtect malware upload',
                uniqueSlug: 'macos-enable-xprotect-malware-upload',
                tooltip: 'If false, prevents Gatekeeper from prompting the user to upload blocked malware to Apple for purposes of improving malware detection.',
                category: 'Gatekeeper',
                payload: 'Control',
                payloadType: 'com.apple.systempolicy.control',
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'EnableXProtectMalwareUpload',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
                },
              },
            ]
          },

        ]
      },
      {
        categoryName: 'Network',
        categorySlug: 'macos-network',
        subcategories: [
          {
            subcategoryName: 'Firewall',
            subcategorySlug: 'macos-firewall',
            description: 'Settings related to the built-in firewall on macOS',
            learnMoreLinkUrl: 'https://developer.apple.com/documentation/devicemanagement/firewall',
            payloads: [
              {
                name: 'Enable firewall',
                uniqueSlug: 'macos-enable-firewall',
                tooltip: 'If true, the system enables the firewall.',
                category: 'Firewall',
                payload: 'Firewall',
                payloadType: 'com.apple.security.firewall',
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'EnableFirewall',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
                },
              },
              {
                name: 'Allow built-in applications',
                uniqueSlug: 'macos-firewall-allow-signed',
                tooltip: 'If true, the system allows built-in software to receive incoming connections. Available in macOS 12.3 and later.',
                category: 'Firewall',
                payload: 'Firewall',
                payloadType: 'com.apple.security.firewall',
                formInput: {
                  type: 'boolean',
                  trueValue: 0,
                  falseValue: 1
                },
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'AllowSigned',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
                },
              },
              {
                name: 'Allow signed applications',
                uniqueSlug: 'macos-firewall-allow-signed-apps',
                tooltip: 'If true, the system allows downloaded signed software to receive incoming connections. Available in macOS 12.3 and later.',
                category: 'Firewall',
                payload: 'Firewall',
                payloadType: 'com.apple.security.firewall',
                formInput: {
                  type: 'boolean',
                  trueValue: 0,
                  falseValue: 1
                },
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'AllowSignedApp',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
                },
              },
              {
                name: 'Block all incoming connections',
                uniqueSlug: 'macos-firewall-block-incoming',
                tooltip: 'If true, the system enables blocking all incoming connections.',
                category: 'Firewall',
                payload: 'Firewall',
                payloadType: 'com.apple.security.firewall',
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'BlockAllIncoming',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
                },
              },
              {
                name: 'Enable stealth mode',
                uniqueSlug: 'macos-firewall-enable-stealth-mode',
                tooltip: 'If true, the system enables stealth mode.',
                category: 'Firewall',
                payload: 'Firewall',
                payloadType: 'com.apple.security.firewall',
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'boolean',
                  settingKey: 'EnableStealthMode',
                  trueValue: '<true/>',
                  falseValue: '<false/>',
                },
              },
              // { TODO: add support for specifying arrays of objects.
              //   name: 'Allow/block specified applications',
              //   uniqueSlug: 'macos-firewall-application-list',
              //   tooltip: 'If true, the system enables stealth mode.',
              //   category: 'Firewall',
              //   payload: 'Firewall',
              //   payloadType: 'com.apple.security.firewall',
              //   formInput: {
              //     type: 'array',
              //     unitLabel: 'Bundle identifier'
              //   },
              //   formOutput: {
              //     settingFormat: 'list',
              //     settingKey: 'Applications',
              //     trueValue: '<true/>',
              //     falseValue: '<false/>',
              //   },
              // },
            ]
          }
        ]
      }
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
                tooltip: 'The number of authentication failures allowed before the device will be wiped. A value of 0 disables device wipe functionality.',
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
                tooltip: `Determines the period of time (in days) that a password can be used before the system requires the user to change it. You can set passwords to expire after a number of days between 1 and 999, or you can specify that passwords never expire by setting the number of days to 0.`,
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
                tooltip: `Determines the period of time (in days) that a password must be used before the user can change it. You can set a value between 1 and 998 days, or you can allow changes immediately by setting the number of days to 0. If the maximum password age is set to 0, the minimum password age can be set to any value between 0 and 998. Configure the minimum password age to be more than 0 if you want Enforce password history to be effective.`,
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
                tooltip: 'The minimum number of characters a device\'s password must be',
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
                tooltip: `The number of complex element types (uppercase and lowercase letters, numbers, and punctuation) required for a strong PIN or password.`,
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
                tooltip: `Specifies whether PINs or passwords such as 1111 or 1234 are allowed. For the desktop, it also controls the use of picture passwords.`,
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
                tooltip: `This security setting determines whether the operating system stores passwords using reversible encryption. Storing passwords using reversible encryption is essentially the same as storing plaintext versions of the passwords. For this reason, this policy should never be enabled unless application requirements outweigh the need to protect password information.`,
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
                tooltip: `Disables the lock screen camera toggle switch in PC Settings and prevents a camera from being invoked on the lock screen.`,
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
                tooltip: `If this policy is enabled, passwords must meet the following minimum requirements:
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
          },
          {
            subcategoryName: 'SmartScreen',
            subcategorySlug: 'windows-smartscreen',
            description: 'Windows Defender SmartScreen provides warning messages to help protect users from potential phishing scams and malicious software.',
            learnMoreLinkUrl: 'https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-smartscreen',
            payloads: [
              {
                name: 'Enable SmartScreen in Microsoft Edge',
                tooltip: `This policy setting lets you configure whether to turn on Windows Defender SmartScreen in Microsoft Edge.`,
                uniqueSlug: 'windows-enable-smartscreen-in-edge',
                category: 'SmartScreen',
                supportedAccessTypes: ['add', 'replace'],
                payloadGroup: 'Microsoft Edge',
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/Browser/AllowSmartScreen',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Allow users to bypass Windows Defender SmartScreen prompts for sites',
                tooltip: `This policy setting lets you configure whether to turn on Windows Defender SmartScreen in Microsoft Edge.`,
                uniqueSlug: 'windows-enable-smartscreen-bypass-in-edge',
                category: 'SmartScreen',
                supportedAccessTypes: ['add', 'replace'],
                payloadGroup: 'Microsoft Edge',
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/Browser/PreventSmartScreenPromptOverride',
                  trueValue: 0,
                  falseValue: 1,
                },
              },
              {
                name: 'Enable SmartScreen in File Explorer',
                tooltip: `Allows IT Admins to configure SmartScreen for Windows.`,
                uniqueSlug: 'windows-enable-smartscreen-in-shell',
                category: 'SmartScreen',
                supportedAccessTypes: ['add', 'replace'],
                payloadGroup: 'File explorer',
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/SmartScreen/EnableSmartScreenInShell',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Allow users to bypass Windows Defender SmartScreen prompts for files',
                tooltip: `Allows IT Admins to control whether users can ignore SmartScreen warnings and run malicious files.`,
                uniqueSlug: 'windows-prevent-override-for-files-in-shell',
                category: 'SmartScreen',
                supportedAccessTypes: ['add', 'replace'],
                payloadGroup: 'File explorer',
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/SmartScreen/PreventOverrideForFilesInShell',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Configure app install control',
                tooltip: `Allows IT Admins to control whether users are allowed to install apps from places other than the Microsoft Store.`,
                uniqueSlug: 'windows-configure-app-install-control',
                category: 'SmartScreen',
                supportedAccessTypes: ['add', 'replace'],
                payloadGroup: 'App installation',
                formInput: {
                  type: 'radio',
                  options: [
                    {
                      name: 'Users can download and install files from anywhere on the web',
                      value: 0
                    },
                    {
                      name: 'Users can only install apps from the Microsoft Store',
                      value: 1
                    },
                    {
                      name: 'Let users know that there\'s a comparable app in the Store',
                      value: 2,
                    },
                    {
                      name: 'Warn users before installing apps from outside the Store',
                      value: 3,
                    }
                  ]
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/SmartScreen/EnableAppInstallControl',
                },
              },

            ]
          },
          {
            subcategoryName: 'BitLocker',
            subcategorySlug: 'windows-bitlocker',
            description: 'Use BitLocker to encrypt drives and protect data on your device.',
            learnMoreLinkUrl: 'https://learn.microsoft.com/en-us/windows/client-management/mdm/bitlocker-csp',
            noteForFleetUsers: 'Disk encryption settings are managed directly in Fleet. Any settings configured here will be ignored.',
            docsLinkForFleetUsers: '/guides/enforce-disk-encryption',
            payloads: [
              {
                name: 'Enable BitLocker for operating system drives',
                uniqueSlug: 'windows-enable-bitlocker-for-os-drives',
                tooltip: 'Require encryption to be turned on using BitLocker.',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/RequireDeviceEncryption',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Enforce encryption type for operating system drives',
                uniqueSlug: 'windows-enforce-encryption-type-for-os-drives',
                tooltip: 'This policy setting allows you to configure the encryption type used by BitLocker Drive Encryption. This policy setting is applied when you turn on BitLocker. Changing the encryption type has no effect if the drive is already encrypted or if encryption is in progress.',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'multifield',
                  inputs: [
                    {
                      type: 'boolean',
                      slug: 'enabled',
                      label: 'Enable',
                      trueValue: '<enabled/>',
                      falseValue: '<enabled/>',
                    },
                    {
                      type: 'radio',
                      label: 'Encryption type',
                      slug: 'encryptionType',
                      options: [
                        {
                          name: 'Allow user to choose encryption type',
                          value: 0
                        },
                        {
                          name: 'Full encryption',
                          value: 1
                        },
                        {
                          name: 'Used space only encryption.',
                          value: 2,
                        },
                      ]
                    },
                  ]
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/SystemDrivesEncryptionType',
                  outputTemplate: `<%= enabled %><data id="OSEncryptionTypeDropDown_Name" value="<%= encryptionType %>"/>`,
                  valuesToTransform: {
                    'enabled': {
                      true: '<enabled/>',
                      false: '<disabled/>',
                    },
                  }
                },
              },
              {
                name: 'Enforce startup authentication',
                uniqueSlug: 'windows-enforce-startup-authentication',
                tooltip: 'This policy setting allows you to configure whether BitLocker requires additional authentication each time the computer starts and whether you are using BitLocker with or without a Trusted Platform Module (TPM).',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'multifield',
                  inputs: [
                    {
                      type: 'boolean',
                      slug: 'enabled',
                      label: 'Enable',
                    },
                    {
                      type: 'select',
                      label: 'TPM startup',
                      slug: 'tpmStartup',
                      options: [
                        {
                          name: 'Disallowed',
                          value: 0
                        },
                        {
                          name: 'Required',
                          value: 1
                        },
                        {
                          name: 'Optional',
                          value: 2,
                        },
                      ]
                    },
                    {
                      type: 'select',
                      label: 'TPM startup key',
                      slug: 'tpmStartupKey',
                      options: [
                        {
                          name: 'Disallowed',
                          value: 0
                        },
                        {
                          name: 'Required',
                          value: 1
                        },
                        {
                          name: 'Optional',
                          value: 2,
                        },
                      ]
                    },
                    {
                      type: 'select',
                      label: 'TPM startup PIN',
                      slug: 'tpmStartupPin',
                      options: [
                        {
                          name: 'Disallowed',
                          value: 0
                        },
                        {
                          name: 'Required',
                          value: 1
                        },
                        {
                          name: 'Optional',
                          value: 2,
                        },
                      ]
                    },
                    {
                      type: 'select',
                      label: 'TPM startup key and PIN',
                      slug: 'tpmStartupKeyAndPin',
                      options: [
                        {
                          name: 'Disallowed',
                          value: 0
                        },
                        {
                          name: 'Required',
                          value: 1
                        },
                        {
                          name: 'Optional',
                          value: 2,
                        },
                      ]
                    },
                    {
                      type: 'booleanWithLabel',
                      slug: 'allowBitlockerWithoutTpm',
                      label: 'Allow BitLocker without a compatible TPM',
                    },
                  ]
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/SystemDrivesRequireStartupAuthentication',
                  outputTemplate:`<%= enabled %><data id="ConfigureNonTPMStartupKeyUsage_Name" value="<%= allowBitlockerWithoutTpm %>"/><data id="ConfigureTPMStartupKeyUsageDropDown_Name" value="<%= tpmStartupKey %>"/><data id="ConfigurePINUsageDropDown_Name" value="<%= tpmStartupPin %>"/><data id="ConfigureTPMPINKeyUsageDropDown_Name" value="<%= tpmStartupKeyAndPin %>"/><data id="ConfigureTPMUsageDropDown_Name" value="<%= tpmStartup %>"/>`,
                  valuesToTransform: {
                    'enabled': {
                      true: '<enabled/>',
                      false: '<disabled/>',
                    },
                  }
                },
              },
              {
                name: 'Enforce enhanced startup PINs',
                uniqueSlug: 'windows-enforce-enhanced-startup-pin',
                tooltip: 'This policy setting allows you to configure whether BitLocker requires additional authentication each time the computer starts and whether you are using BitLocker with or without a Trusted Platform Module (TPM).',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/SystemDrivesEnhancedPIN',
                  trueValue: '<enabled/>',
                  falseValue: '<disabled/>',
                },
              },
              {
                name: 'Enforce recovery options for operating system drives',
                uniqueSlug: 'windows-enforce-system-drive-recovery-options',
                tooltip: 'This policy setting allows you to control how BitLocker-protected operating system drives are recovered in the absence of the required startup key information. This policy setting is applied when you turn on BitLocker.',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'multifield',
                  inputs: [
                    {
                      type: 'boolean',
                      slug: 'enabled',
                      label: 'Enable',
                    },
                    {
                      type: 'select',
                      label: 'Configure 256-bit recovery key',
                      slug: 'recoveryKey',
                      options: [
                        {
                          name: 'Disallowed',
                          value: 0
                        },
                        {
                          name: 'Required',
                          value: 1
                        },
                        {
                          name: 'Allowed',
                          value: 2,
                        },
                      ]
                    },
                    {
                      type: 'select',
                      label: 'Configure 48-digit recovery password',
                      slug: 'recoveryPassword',
                      options: [
                        {
                          name: 'Disallowed',
                          value: 0
                        },
                        {
                          name: 'Required',
                          value: 1
                        },
                        {
                          name: 'Allowed',
                          value: 2,
                        },
                      ]
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Store BitLocker recovery information on Active Directory',
                      slug: 'storeOnActiveDirectory',
                    },
                    {
                      type: 'select',
                      label: 'Choose what recovery information to store on Active Directory',
                      slug: 'whatToStoreOnActiveDirectory',
                      options: [
                        {
                          name: 'Store recovery passwords and key packages.',
                          value: 1
                        },
                        {
                          name: 'Store recovery passwords only.',
                          value: 2,
                        },
                      ]
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Do not enable BitLocker until recovery information is stored to Active Directory',
                      slug: 'doNotEnableUntilStored',
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Allow data recovery agent',
                      slug: 'allowDataRecoveryAgent',
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Omit recovery options from BitLocker setup wizard',
                      slug: 'hideRecoveryPage',
                    },
                  ]
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/SystemDrivesRecoveryOptions',
                  outputTemplate:`<%= enabled %><data id="OSAllowDRA_Name" value="<%= allowDataRecoveryAgent %>"/><data id="OSRecoveryPasswordUsageDropDown_Name" value="<%= recoveryPassword %>"/><data id="OSRecoveryKeyUsageDropDown_Name" value="<%= recoveryKey %>"/><data id="OSHideRecoveryPage_Name" value="<%= hideRecoveryPage %>"/><data id="OSActiveDirectoryBackup_Name" value="<%= storeOnActiveDirectory %>"/><data id="OSActiveDirectoryBackupDropDown_Name" value="<%= whatToStoreOnActiveDirectory %>"/><data id="OSRequireActiveDirectoryBackup_Name" value="<%= doNotEnableUntilStored %>"/>`,
                  valuesToTransform: {
                    'enabled': {
                      true: '<enabled/>',
                      false: '<disabled/>',
                    },
                  }
                },
              },
              {
                name: 'Enable BitLocker for fixed data drives',
                uniqueSlug: 'windows-enable-bitlocker-for-fixed-data-drives',
                tooltip: 'This policy setting determines whether BitLocker protection is required for fixed data drives to be writable on a computer.',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/FixedDrivesRequireEncryption',
                  trueValue: '<enabled/>',
                  falseValue: '<disabled/>',
                },
              },
              {
                name: 'Enforce encryption type for fixed data drives',
                uniqueSlug: 'windows-enforce-encryption-type-for-fixed-data-drives',
                tooltip: 'This policy setting allows you to configure the encryption type used by BitLocker Drive Encryption. This policy setting is applied when you turn on BitLocker. Changing the encryption type has no effect if the drive is already encrypted or if encryption is in progress. ',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'multifield',
                  inputs: [
                    {
                      type: 'boolean',
                      slug: 'enabled',
                      label: 'Enable',
                      trueValue: '<enabled/>',
                      falseValue: '<enabled/>',
                    },
                    {
                      type: 'radio',
                      label: 'Encryption type',
                      slug: 'encryptionType',
                      options: [
                        {
                          name: 'Allow user to choose encryption type',
                          value: 0
                        },
                        {
                          name: 'Full encryption',
                          value: 1
                        },
                        {
                          name: 'Used space only encryption.',
                          value: 2,
                        },
                      ]
                    },
                  ]
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/FixedDrivesEncryptionType',
                  outputTemplate: `<%= enabled %><data id="FDVEncryptionTypeDropDown_Name" value="<%= encryptionType %>"/>`,
                  valuesToTransform: {
                    'enabled': {
                      true: '<enabled/>',
                      false: '<disabled/>',
                    },
                  }
                },
              },
              {
                name: 'Enforce recovery options for operating system drives',
                uniqueSlug: 'windows-enforce-fixed-data-drive-recovery-options',
                tooltip: 'This policy setting allows you to control how BitLocker-protected fixed data drives are recovered in the absence of the required credentials. This policy setting is applied when you turn on BitLocker.',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'multifield',
                  inputs: [
                    {
                      type: 'boolean',
                      slug: 'enabled',
                      label: 'Enable',
                    },
                    {
                      type: 'select',
                      label: 'Configure 256-bit recovery key',
                      slug: 'recoveryKey',
                      options: [
                        {
                          name: 'Disallowed',
                          value: 0
                        },
                        {
                          name: 'Required',
                          value: 1
                        },
                        {
                          name: 'Allowed',
                          value: 2,
                        },
                      ]
                    },
                    {
                      type: 'select',
                      label: 'Configure 48-digit recovery password',
                      slug: 'recoveryPassword',
                      options: [
                        {
                          name: 'Disallowed',
                          value: 0
                        },
                        {
                          name: 'Required',
                          value: 1
                        },
                        {
                          name: 'Allowed',
                          value: 2,
                        },
                      ]
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Store BitLocker recovery information on Active Directory',
                      slug: 'storeOnActiveDirectory',
                    },
                    {
                      type: 'select',
                      label: 'Choose what recovery information to store on Active Directory',
                      slug: 'whatToStoreOnActiveDirectory',
                      options: [
                        {
                          name: 'Store recovery passwords and key packages.',
                          value: 1
                        },
                        {
                          name: 'Store recovery passwords only.',
                          value: 2,
                        },
                      ]
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Do not enable BitLocker until recovery information is stored to Active Directory',
                      slug: 'doNotEnableUntilStored',
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Allow data recovery agent',
                      slug: 'allowDataRecoveryAgent',
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Omit recovery options from BitLocker setup wizard',
                      slug: 'hideRecoveryPage',
                    },
                  ]
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/FixedDrivesRecoveryOptions',
                  outputTemplate:`<%= enabled %><data id="FDVAllowDRA_Name" value="<%= allowDataRecoveryAgent %>"/><data id="FDVRecoveryPasswordUsageDropDown_Name" value="<%= recoveryPassword %>"/><data id="FDVRecoveryKeyUsageDropDown_Name" value="<%= recoveryKey %>"/><data id="FDVHideRecoveryPage_Name" value="<%= hideRecoveryPage %>"/><data id="FDVActiveDirectoryBackup_Name" value="<%= storeOnActiveDirectory %>"/><data id="FDVActiveDirectoryBackupDropDown_Name" value="<%= whatToStoreOnActiveDirectory %>"/><data id="FDVRequireActiveDirectoryBackup_Name" value="<%= doNotEnableUntilStored %>"/>`,
                  valuesToTransform: {
                    'enabled': {
                      true: '<enabled/>',
                      false: '<disabled/>',
                    },
                  }
                },
              },
              {
                name: 'Deny write access to fixed data drives not protected by BitLocker',
                uniqueSlug: 'windows-deny-wrtie-access-to-fixed-data-drives',
                tooltip: 'This policy setting determines whether BitLocker protection is required for fixed data drives to be writable on a computer. If you enable this policy setting, all fixed data drives that aren\'t BitLocker-protected will be mounted as read-only.',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/FixedDrivesRequireEncryption',
                  trueValue: '<enabled/>',
                  falseValue: '<disabled/>',
                },
              },
              {
                name: 'Enable BitLocker for removable data drives',
                uniqueSlug: 'windows-enable-bitlocker-for-removeable-data-drives',
                tooltip: 'This policy setting controls the use of BitLocker on removable data drives.',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'multifield',
                  inputs: [
                    {
                      type: 'boolean',
                      slug: 'enabled',
                      label: 'Enable',
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Allow users to apply BitLocker protection on removable data drives',
                      slug: 'allowApplyBitlocker',
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Allow users to suspend and decrypt BitLocker on removable data drives',
                      slug: 'allowDisableBitlocker',
                    },
                  ],
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/RemovableDrivesConfigureBDE',
                  outputTemplate:`<%= enabled %><data id="RDVAllowBDE_Name" value="<%= allowApplyBitlocker %>"/><data id="RDVDisableBDE_Name" value="<%= allowDisableBitlocker %>"/>`,
                  valuesToTransform: {
                    'enabled': {
                      true: '<enabled/>',
                      false: '<disabled/>',
                    },
                  }
                },
              },
              {
                name: 'Enforce encryption type for removable data drives',
                uniqueSlug: 'windows-enforce-encryption-type-for-removeable-drives',
                tooltip: 'This policy setting allows you to configure the encryption type used by BitLocker Drive Encryption. This policy setting is applied when you turn on BitLocker. Changing the encryption type has no effect if the drive is already encrypted or if encryption is in progress. ',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'multifield',
                  inputs: [
                    {
                      type: 'boolean',
                      slug: 'enabled',
                      label: 'Enable',
                      trueValue: '<enabled/>',
                      falseValue: '<enabled/>',
                    },
                    {
                      type: 'radio',
                      label: 'Encryption type',
                      slug: 'encryptionType',
                      options: [
                        {
                          name: 'Allow user to choose encryption type',
                          value: 0
                        },
                        {
                          name: 'Full encryption',
                          value: 1
                        },
                        {
                          name: 'Used space only encryption.',
                          value: 2,
                        },
                      ]
                    },
                  ]
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/RemovableDrivesEncryptionType',
                  outputTemplate: `<%= enabled %><data id="RDVEncryptionTypeDropDown_Name" value="<%= encryptionType %>"/>`,
                  valuesToTransform: {
                    'enabled': {
                      true: '<enabled/>',
                      false: '<disabled/>',
                    },
                  }
                },
              },
              {
                name: 'Deny write access to removable data drives not protected by BitLocker',
                uniqueSlug: 'windows-deny-write-access-to-removeable-data-drives',
                tooltip: 'This policy setting configures whether BitLocker protection is required for a computer to be able to write data to a removable data drive.',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'multifield',
                  inputs: [
                    {
                      type: 'boolean',
                      slug: 'enabled',
                      label: 'Enable',
                    },
                    {
                      type: 'booleanWithLabel',
                      label: 'Deny write access to devices configured in another organization',
                      slug: 'crossOrg',
                    },
                  ]
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/RemovableDrivesRequireEncryption',
                  outputTemplate: `<%= enabled %><data id="RDVCrossOrg" value="<%= crossOrg %>"/>`,
                  valuesToTransform: {
                    'enabled': {
                      true: '<enabled/>',
                      false: '<disabled/>',
                    },
                  }
                },
              },
              {
                name: 'Enforce encryption method',
                uniqueSlug: 'windows-enforce-encryption-method',
                tooltip: 'This policy setting configures whether BitLocker protection is required for a computer to be able to write data to a removable data drive.',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'multifield',
                  inputs: [
                    {
                      type: 'boolean',
                      slug: 'enabled',
                      label: 'Enable',
                    },
                    {
                      type: 'select',
                      label: 'Operating system drives',
                      slug: 'osEncryptionType',
                      options: [
                        {
                          name: 'AES-CBC 128',
                          value: 3
                        },
                        {
                          name: 'AES-CBC 256',
                          value: 4,
                        },
                        {
                          name: 'XTS-AES 128',
                          value: 6,
                        },
                        {
                          name: 'XTS-AES 256',
                          value: 7,
                        },
                      ]
                    },
                    {
                      type: 'select',
                      label: 'Fixed data drives',
                      slug: 'fixedDriveEncryptionType',
                      options: [
                        {
                          name: 'AES-CBC 128',
                          value: 3
                        },
                        {
                          name: 'AES-CBC 256',
                          value: 4,
                        },
                        {
                          name: 'XTS-AES 128',
                          value: 6,
                        },
                        {
                          name: 'XTS-AES 256',
                          value: 7,
                        },
                      ]
                    },
                    {
                      type: 'select',
                      label: 'Removable data drives',
                      slug: 'removeableDriveEncryptionType',
                      options: [
                        {
                          name: 'AES-CBC 128',
                          value: 3
                        },
                        {
                          name: 'AES-CBC 256',
                          value: 4,
                        },
                        {
                          name: 'XTS-AES 128',
                          value: 6,
                        },
                        {
                          name: 'XTS-AES 256',
                          value: 7,
                        },
                      ]
                    },
                  ]
                },
                formOutput: {
                  settingFormat: 'chr',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/EncryptionMethodByDriveType',
                  outputTemplate: `<%= enabled %><data id="EncryptionMethodWithXtsOsDropDown_Name" value="<%= osEncryptionType %>"/><data id="EncryptionMethodWithXtsFdvDropDown_Name" value="<%= fixedDriveEncryptionType %>"/><data id="EncryptionMethodWithXtsRdvDropDown_Name" value="<%= removeableDriveEncryptionType %>"/>`,
                  valuesToTransform: {
                    'enabled': {
                      true: '<enabled/>',
                      false: '<disabled/>',
                    },
                  }
                },
              },
              {
                name: 'Configure recovery password rotation',
                uniqueSlug: 'windows-configure-recover-password-roration',
                tooltip: 'Allows Admin to configure Numeric Recovery Password Rotation upon use for OS and fixed drives on Entra ID and hybrid domain joined devices.',
                category: 'BitLocker',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'radio',
                  label: 'Encryption type',
                  options: [
                    {
                      name: 'Disable password rotation',
                      value: 0
                    },
                    {
                      name: 'Enable password rotation for Azure AD-joined devices',
                      value: 1
                    },
                    {
                      name: 'Enable password rotation for Azure AD-joined and hybrid-joined devices',
                      value: 2,
                    },
                  ]
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/BitLocker/ConfigureRecoveryPasswordRotation',
                },
              },
            ],
          }
        ]
      },
      {
        categoryName: 'Software & updates',
        categorySlug: 'windows-software-and-updates',
        subcategories: [
          {
            subcategoryName: 'Applications',
            subcategorySlug: 'windows-applications',
            description: 'Settings related to applications and the Windows store.',
            learnMoreLinkUrl: 'https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-applicationmanagement',
            payloads: [
              {
                name: 'Allow all trusted apps',
                tooltip: `This policy setting allows you to manage the installation of trusted line-of-business (LOB) or developer-signed packaged Microsoft Store apps.
                If you enable this policy setting, you can install any LOB or developer-signed packaged Microsoft Store app (which must be signed with a certificate chain that can be successfully validated by the local computer).

                If you disable or don't configure this policy setting, you can't install LOB or developer-signed packaged Microsoft Store apps.`,
                uniqueSlug: 'windows-allow-all-trusted-apps',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/AllowAllTrustedApps',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Allow automatic updates for Microsoft store apps',
                tooltip: 'Specifies whether automatic update of apps from Microsoft Store are allowed.',
                uniqueSlug: 'windows-allow-app-store-auto-updates',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/AllowAppStoreAutoUpdate',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Allow automatic app archiving',
                tooltip: `This policy setting controls whether the system can archive infrequently used apps.

                If you enable this policy setting, then the system will periodically check for and archive infrequently used apps.

                If you disable this policy setting, then the system won't archive any apps.

                If you don't configure this policy setting (default), then the system will follow default behavior, which is to periodically check for and archive infrequently used apps, and the user will be able to configure this setting themselves.`,
                uniqueSlug: 'windows-allow-automatic-app-archiving',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/AllowAutomaticAppArchiving',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Allow developer unlock',
                tooltip: `Allows or denies development of Microsoft Store applications and installing them directly from an IDE. If you enable this setting and enable the "Allow all trusted apps to install" Group Policy, you can develop Microsoft Store apps and install them directly from an IDE. If you disable or don't configure this setting, you can't develop Microsoft Store apps or install them directly from an IDE. `,
                uniqueSlug: 'windows-allow-developer-unlock',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/AllowDeveloperUnlock',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Allow shared user app data',
                tooltip: `Manages a Windows app's ability to share data between users who have installed the app.
                If you enable this policy, a Windows app can share app data with other instances of that app. Data is shared through the SharedLocal folder. This folder is available through the Windows.Storage API.

                If you disable this policy, a Windows app can't share app data with other instances of that app. If this policy was previously enabled, any previously shared app data will remain in the SharedLocal folder.`,
                uniqueSlug: 'windows-allow-shared-user-app-data',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/AllowSharedUserAppData',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Block app installation for non-admin users',
                tooltip: `Manages non-Administrator users' ability to install Windows app packages.
                If enabled, non-Administrators will be unable to initiate installation of Windows app packages. Administrators who wish to install an app will need to do so from an Administrator context
                If disabled or not confgiured, all users will be able to initiate installation of Windows app packages.`,
                uniqueSlug: 'windows-block-non-admin-user-install',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/BlockNonAdminUserInstall',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Disable Microsoft store originated applications',
                tooltip: `Disable turns off the launch of all apps from the Microsoft Store that came pre-installed or were downloaded. The Microsoft Store will also be disabled.`,
                uniqueSlug: 'windows-disabled-store-originated-apps',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/DisableStoreOriginatedApps',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Allow user control of installation options',
                tooltip: 'This policy setting permits users to change installation options that typically are available only to system administrators.',
                uniqueSlug: 'windows-allow-user-control-over-install',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/MSIAllowUserControlOverInstall',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Always install with elevated privleges',
                tooltip: 'This policy setting directs Windows Installer to use elevated permissions when it installs any program on the system.',
                uniqueSlug: 'windows-always-install-with-elevated-permissions',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/MSIAlwaysInstallWithElevatedPrivileges',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Restrict application data to system volume',
                tooltip: `Prevent users' app data from moving to another location when an app is moved or installed on another location.`,
                uniqueSlug: 'windows-restrict-app-data-to-system-volume',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/RestrictAppDataToSystemVolume',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
              {
                name: 'Restrict applications to system volume',
                tooltip: 'This policy setting allows you to manage installing Windows apps on additional volumes such as secondary partitions, USB drives, or SD cards.',
                uniqueSlug: 'windows-restrict-apps-to-system-volume',
                category: 'Applications',
                supportedAccessTypes: ['add', 'replace'],
                formInput: {
                  type: 'boolean',
                },
                formOutput: {
                  settingFormat: 'int',
                  settingTarget: './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/RestrictAppToSystemVolume',
                  trueValue: 1,
                  falseValue: 0,
                },
              },
            ]
          }
        ],
      }
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
    // Platform select form submission
    handleSubmittingPlatformSelectForm: async function() {
      this.selectedPlatform = this.platformSelectFormData.platform;
      this.step = 'configuration-builder';
    },
    typeFilterSettings: async function() {
      // TODO.
    },
    // Controls which category is expanded in the left sidebar.
    clickExpandCategory: function(category) {
      this.expandedCategory = category;
    },
    // Download modal form submission
    handleSubmittingDownloadProfileForm: async function() {
      this.syncing = true;
      if(this.currentSelectedCategoryForDownload) {
        // this.currentSelectedCategoryForDownload = undefined;
        if(this.selectedPlatform === 'windows') {
          await this.buildWindowsProfile(this.selectedOptionsInAPayload);
        } else if(this.selectedPlatform === 'macos') {
          await this.buildMacOSProfile(this.selectedOptionsInAPayload);
        }
      } else {
        if(this.selectedPlatform === 'windows') {
          await this.buildWindowsProfile(this.selectedPayloads);
        } else if(this.selectedPlatform === 'macos') {
          await this.buildMacOSProfile(this.selectedPayloads);
        }
      }
    },
    // Download modal form submission (single payload)
    handleSubmittingSinglePayloadDownloadProfileForm: async function() {
      this.syncing = true;

      if(this.selectedPlatform === 'windows') {
        await this.buildWindowsProfile(this.selectedPayloads);
      } else if(this.selectedPlatform === 'macos') {
        await this.buildMacOSProfile(this.selectedPayloads);
      }
    },
    buildWindowsProfile: function(payloadsToUse) {
      let xmlString = '';
      // Iterate through the selcted payloads
      for(let payload of payloadsToUse) {
        let payloadToAdd = _.clone(payload);
        // Get the selected access type for this payload
        let accessType = this.configurationBuilderFormData[payload.uniqueSlug+'-access-type'];
        let value;
        if(payload.formInput.type === 'multifield') {
          // If the payload's formInput type is multifield, we'll need to combine the values for this payload.
          // build a dictionary of formData where each key is the input's slug.
          if(!payload.formOutput.outputTemplate){
            throw new Error('Consistency violation, a multifield form input is missing a template value', payload);
          }
          let formDataForThisPayload = {};
          for(let input of payload.formInput.inputs) {
            if(payload.formOutput.valuesToTransform && payload.formOutput.valuesToTransform[input.slug]){
              formDataForThisPayload[input.slug] = payload.formOutput.valuesToTransform[input.slug][this.configurationBuilderFormData[payload.uniqueSlug+'-'+input.slug]];
            } else {
              formDataForThisPayload[input.slug] = this.configurationBuilderFormData[payload.uniqueSlug+'-'+input.slug];
            }
          }
          // Now we'll pass the formData into the formOutput's template string.
          let templateToUse = _.template(payload.formOutput.outputTemplate);
          value = _.trim(templateToUse(formDataForThisPayload));
        } else {
          // Get the selected value for this payload
          value = this.configurationBuilderFormData[payload.uniqueSlug+'-value'];
          // If this payload is a boolean input, we'll convert the true/false value into the expected value for this payload.
          if(payload.formInput.type === 'boolean'){
            if(value) {
              value = payload.formOutput.trueValue;
            } else {
              value = payload.formOutput.falseValue;
            }
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
    buildMacOSProfile: function(selectedPayloads) {
      let xmlString = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
<key>PayloadContent</key>
<array>
`;
      let payloadstoUse = _.clone(selectedPayloads);
      // Iterate through the selcted payloads
      // group selected payloads by their payload type value.
      let payloadsToCreateDictonariesFor = _.groupBy(payloadstoUse, 'payloadType');
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
    // When users click the "remove all" button at the top of the payload card.
    clickRemoveOneCategoryPayloadOptions: function(category) {
      let optionsToRemove = this.selectedPayloadsGroupedByCategory[category];
      this.selectedPayloadsGroupedByCategory = _.without(this.selectedPayloadsGroupedByCategory, category);
      for(let option of optionsToRemove){
        if(option.formInput.type === 'multifield') {
          for(let input of option.formInput.inputs){
            delete this.configurationBuilderFormRules[option.uniqueSlug+'-'+input.slug];
            delete this.configurationBuilderFormData[option.uniqueSlug+'-'+input.slug];
          }
        } else{
          delete this.configurationBuilderFormRules[option.uniqueSlug+'-value'];
          delete this.configurationBuilderFormData[option.uniqueSlug+'-value'];
        }
        let newSelectedPayloads = _.without(this.selectedPayloads, option);
        this.selectedPayloadSettings[option.uniqueSlug] = false;
        this.selectedPayloads = _.uniq(newSelectedPayloads);
        if(this.selectedPlatform === 'windows') {
          delete this.configurationBuilderFormRules[option.uniqueSlug+'-access-type'];
          delete this.configurationBuilderFormData[option.uniqueSlug+'-access-type'];
        }
      }
      this.selectedPayloadsGroupedByCategory = _.groupBy(this.selectedPayloads, 'category');
    },
    // When users click the "remove" button under a single payload option.
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
    // When users click the download all button.
    handleSubmittingConfigurationBuilderForm: function() {
      if(_.keysIn(this.selectedPayloadsGroupedByCategory).length > 1) {
        // If there is more than one payload in this profile, show a warning in a modal.
        this.modal = 'multiple-payloads-selected';
      } else {
        this.openDownloadModal();
      }
    },
    // When users click the downlaod button on a paylaod card.
    handleSubmittingSinglePayloadConfigurationBuilderForm: function() {
      let payloadsInThisCategory = _.filter(this.selectedPayloads, (payload)=>{
        return payload.category === this.currentSelectedCategoryForDownload;
      });
      this.selectedOptionsInAPayload = payloadsInThisCategory;
      this.openDownloadModal();
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
    clickSetCurrentSelectedCategory: async function(category) {
      this.currentSelectedCategoryForDownload = category;
      // console.log(category);
    },
    clickSelectPayload: async function(payloadSlug) {
      if(!this.selectedPayloadSettings[payloadSlug]){
        let selectedPayload = _.find(this.selectedPayloadCategory.payloads, {uniqueSlug: payloadSlug}) || {};
        if(!this.configurationBuilderByCategoryFormRules[selectedPayload.category]) {
          this.configurationBuilderByCategoryFormRules[selectedPayload.category] = {};
        }
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
        if(selectedPayload.formInput.type === 'multifield') {
          for(let input of selectedPayload.formInput.inputs){
            this.configurationBuilderFormRules[selectedPayload.uniqueSlug+'-'+input.slug] = {required: true};
            this.configurationBuilderByCategoryFormRules[selectedPayload.category][selectedPayload.uniqueSlug+'-'+input.slug] = {required: true};
            if(input.type === 'boolean' || input.type === 'booleanWithLabel'){
              // default boolean inputs to false.
              this.configurationBuilderFormData[selectedPayload.uniqueSlug+'-'+input.slug] = false;
            } else if(input.type === 'number') {
              this.configurationBuilderFormData[selectedPayload.uniqueSlug+'-'+input.slug] = input.defaultValue;
            }

          }
        } else {
          this.configurationBuilderFormRules[selectedPayload.uniqueSlug+'-value'] = {required: true};
          this.configurationBuilderByCategoryFormRules[selectedPayload.category][selectedPayload.uniqueSlug+'-value'] = {required: true};
          if(selectedPayload.formInput.type === 'boolean'){
            // default boolean inputs to false.
            this.configurationBuilderFormData[selectedPayload.uniqueSlug+'-value'] = false;
          } else if(selectedPayload.formInput.type === 'number') {
            this.configurationBuilderFormData[selectedPayload.uniqueSlug+'-value'] = selectedPayload.formInput.defaultValue;
          }
        }

        if(this.selectedPlatform === 'windows') {
          this.configurationBuilderFormRules[selectedPayload.uniqueSlug+'-access-type'] = {required: true};
          this.configurationBuilderByCategoryFormRules[selectedPayload.category][selectedPayload.uniqueSlug+'-access-type'] = {required: true};
        }
        this.selectedPayloadsGroupedByCategory = _.groupBy(this.selectedPayloads, 'category');
        this.selectedPayloadSettings[payloadSlug] = true;
        // console.log(this.configurationBuilderFormData);
      } else {
        // Remove the payload option and all dependencies
        let payloadToRemove = _.find(this.selectedPayloads, {uniqueSlug: payloadSlug});
        // check the alsoAutoSetWhenSelected value of the payload we're removing.
        let newSelectedPayloads = _.without(this.selectedPayloads, payloadToRemove);
        if(payloadToRemove.formInput.type === 'multifield') {
          for(let input of payloadToRemove.formInput.inputs){
            delete this.configurationBuilderFormRules[payloadToRemove.uniqueSlug+'-'+input.slug];
            delete this.configurationBuilderFormData[payloadToRemove.uniqueSlug+'-'+input.slug];
          }
        } else {
          delete this.configurationBuilderFormRules[payloadToRemove.uniqueSlug+'-value'];
          delete this.configurationBuilderFormData[payloadToRemove.uniqueSlug+'-value'];
        }
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

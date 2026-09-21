module.exports = {


  friendlyName: 'Get configuration profile generator configuration',


  description: 'Builds and returns the prompts and configuration that the configuration profile generator and related test script uses.',



  inputs: {
    profileType: {
      type: 'string',
      required: true,
      isIn: [
        'mobileconfig',
        'ddm',
        'csp',
      ],
    },
    naturalLanguageInstructions: {
      type: 'string',
      required: true,
      description: 'What the IT admin asked for, in their own words.'
    },

    useLighterResponseShape: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Whether or not to request less information back with the generated profile.'
    }
  },


  exits: {

    success: {
      outputFriendlyName: 'Configuration profile generator configuration',
    },

  },


  fn: async function ({profileType, naturalLanguageInstructions, useLighterResponseShape}) {

    // Apple's published DDM configuration declarations, pruned to what a generator must not get wrong:
    // exact key name, type, and whatever constrains the value.  Titles, prose and per-OS availability are
    // dropped.  Regenerate by walking apple/device-management's declarative/declarations/configurations.
    const DDM_DECLARATION_SCHEMA_V1 = `com.apple.configuration.account.caldav
      VisibleName:string, HostName:string*, Port:integer, Path:string, AuthenticationCredentialsAssetReference:string
    com.apple.configuration.account.carddav
      VisibleName:string, HostName:string*, Port:integer, Path:string, AuthenticationCredentialsAssetReference:string
    com.apple.configuration.account.exchange
      VisibleName:string, EnabledProtocolTypes[], UserIdentityAssetReference:string, HostName:string, Port:integer, Path:string, ExternalHostName:string, ExternalPort:integer, External Path:string, OAuth{Enabled:boolean*, SignInURL:string, TokenRequestURL:string}, AuthenticationCredentialsAssetReference:string, AuthenticationIdentityAssetReference:string, SMIME{Signing{Enabled:boolean*, IdentityAssetReference:string, UserOverrideable:boolean, IdentityUserOverrideable:boolean}, Encryption{Enabled:boolean*, IdentityAssetReference:string, UserOverrideable:boolean, IdentityUserOverrideable:boolean, PerMessageSwitchEnabled:boolean}}, MailServiceActive:boolean, LockMailService:boolean, ContactsServiceActive:boolean, LockContactsService:boolean, CalendarServiceActive:boolean, LockCalendarService:boolean, RemindersServiceActive:boolean, LockRemindersService:boolean, NotesServiceActive:boolean, LockNotesService:boolean
    com.apple.configuration.account.google
      VisibleName:string, UserIdentityAssetReference:string*
    com.apple.configuration.account.ldap
      VisibleName:string, HostName:string*, Port:integer, AuthenticationCredentialsAssetReference:string, SearchSettings[]
    com.apple.configuration.account.mail
      VisibleName:string, UserIdentityAssetReference:string, IncomingServer{ServerType:string(IMAP|POP)*, HostName:string*, Port:integer, AuthenticationMethod:string(None|Password|CRAMMD5|NTLM|HTTPMD5)*, AuthenticationCredentialsAssetReference:string, IMAPPathPrefix:string}, OutgoingServer{HostName:string*, Port:integer, AuthenticationMethod:string(None|Password|CRAMMD5|NTLM|HTTPMD5)*, AuthenticationCredentialsAssetReference:string}, SMIME{Signing{Enabled:boolean*, IdentityAssetReference:string, UserOverrideable:boolean, IdentityUserOverrideable:boolean}, Encryption{Enabled:boolean*, IdentityAssetReference:string, UserOverrideable:boolean, IdentityUserOverrideable:boolean, PerMessageSwitchEnabled:boolean}}
    com.apple.configuration.account.subscribed-calendar
      VisibleName:string, CalendarURL:string*, AuthenticationCredentialsAssetReference:string
    com.apple.configuration.app.managed
      AppStoreID:string, BundleID:string, ManifestURL:string, AppComposedIdentifier:string, iOSApp:boolean, InstallBehavior{Install:string(Optional|Required), License{Assignment:string(Device|User), VPPType:string(Device|User)}, Version:integer, AllowDownloadsOverCellular:string(AlwaysOn|AlwaysOff|StoreSettings)}, UpdateBehavior{AutomaticAppUpdates:string(AlwaysOn|AlwaysOff|StoreSettings)*}, IncludeInBackup:boolean, Attributes{AssociatedDomains[], AssociatedDomainsEnableDirectDownloads:boolean, CellularSliceUUID:string, ContentFilterUUID:string, DNSProxyUUID:string, Hideable:boolean, Lockable:boolean, RelayUUID:string, TapToPayScreenLock:boolean, VPNUUID:string}, AppConfig{DataAssetReference:string, Passwords[], Identities[], Certificates[]}, ExtensionConfigs{ANY{DataAssetReference:string, Passwords[], Identities[], Certificates[]}}, LegacyAppConfigAssetReference:string
    com.apple.configuration.audio-accessory.settings
      TemporaryPairing{Disabled:boolean, Configuration{UnpairingTime{Policy:string(None|Hour)*, Hour:integer(0-23)}}}
    com.apple.configuration.diskmanagement.settings
      Restrictions{ExternalStorage:string(Allowed|ReadOnly|Disallowed), NetworkStorage:string(Allowed|ReadOnly|Disallowed)}
    com.apple.configuration.external-intelligence.settings
      Enabled:boolean, AllowSignIn:boolean, AllowedWorkspaceIDs[]
    com.apple.configuration.intelligence.settings
      AllowAppleIntelligenceReport:boolean, AllowGenmoji:boolean, AllowImagePlayground:boolean, AllowImageWand:boolean, AllowPersonalizedHandwritingResults:boolean, AllowVisualIntelligenceSummary:boolean, AllowWritingTools:boolean, Apps{Mail{AllowSmartReplies:boolean, AllowSummary:boolean}, Notes{AllowTranscription:boolean, AllowTranscriptionSummary:boolean}, Safari{AllowSummary:boolean}}, ForceOnDeviceOnlyDictation:boolean, ForceOnDeviceOnlyTranslation:boolean
    com.apple.configuration.keyboard.settings
      AllowAutoCorrection:boolean, AllowDefinitionLookup:boolean, AllowDictation:boolean, AllowMathKeyboardSuggestions:boolean, AllowPredictiveText:boolean, AllowSlideToType:boolean, AllowSpellCheck:boolean, AllowTextReplacement:boolean
    com.apple.configuration.legacy
      ProfileURL:string*
    com.apple.configuration.legacy.interactive
      ProfileURL:string*, VisibleName:string*
    com.apple.configuration.management.status-subscriptions
      StatusItems[]
    com.apple.configuration.management.test
      Echo:string*, EchoDataAssetReference:string, ReturnStatus:string(Installed|Failed|Unlocked)
    com.apple.configuration.math.settings
      Calculator{BasicMode{AddSquareRoot:boolean*}, ScientificMode{Enabled:boolean*}, ProgrammerMode{Enabled:boolean*}, MathNotesMode{Enabled:boolean*}, InputModes{UnitConversion:boolean*, RPN:boolean*}}, SystemBehavior{KeyboardSuggestions:boolean*, MathNotes:boolean*}
    com.apple.configuration.migration-assistant.settings
      ShouldDoManagedMigration:boolean*, ExcludedAccounts[], ExcludedPaths[], RequiredPaths[], ShouldMigrateSecurityPrivacySettings:boolean*
    com.apple.configuration.package
      ManifestURL:string*, InstallBehavior{Install:string(Optional|Required)}
    com.apple.configuration.passcode.settings
      RequirePasscode:boolean, RequireAlphanumericPasscode:boolean, RequireComplexPasscode:boolean, MinimumLength:integer(0-16), MinimumComplexCharacters:integer(0-4), MaximumFailedAttempts:integer(2-11), FailedAttemptsResetInMinutes:integer, MaximumGracePeriodInMinutes:integer, MaximumInactivityInMinutes:integer(0-15), MaximumPasscodeAgeInDays:integer(0-730), PasscodeReuseLimit:integer(1-50), ChangeAtNextAuth:boolean, CustomRegex{Regex:string*, Description{ANY:string}}
    com.apple.configuration.safari.bookmarks
      ManagedBookmarks[]
    com.apple.configuration.safari.extensions.settings
      ManagedExtensions{ANY{State:string(Allowed|AlwaysOn|AlwaysOff), PrivateBrowsing:string(Allowed|AlwaysOn|AlwaysOff), AllowedDomains[], DeniedDomains[]}}
    com.apple.configuration.safari.settings
      AcceptCookies:string(Never|CurrentWebsite|VisitedWebsites|Always), AllowDisablingFraudWarning:boolean, AllowHistoryClearing:boolean, AllowJavaScript:boolean, AllowPrivateBrowsing:boolean, AllowPopups:boolean, AllowSummary:boolean, NewTabStartPage{PageType:string(Start|Home|Extension)*, HomepageURL:string, ExtensionIdentifier:string}
    com.apple.configuration.screensharing.connection
      ConnectionUUID:string*, DisplayName:string*, HostName:string*, Port:integer, DisplayConfiguration{DisplayType:string(Virtual1|Virtual2)*}, AuthenticationCredentialsAssetReference:string
    com.apple.configuration.screensharing.connection.group
      ConnectionGroupUUID:string*, GroupName:string*, Members[]
    com.apple.configuration.screensharing.host.settings
      MaximumVirtualDisplays:integer(0-2), PortBase:integer(1024-65535), PreventCopyFilesFromHost:boolean, PreventCopyFilesToHost:boolean, PreventHighPerformanceConnections:boolean
    com.apple.configuration.security.certificate
      CredentialAssetReference:string*
    com.apple.configuration.security.identity
      CredentialAssetReference:string*, AllowAllAppsAccess:boolean, KeyIsExtractable:boolean
    com.apple.configuration.security.passkey.attestation
      AttestationIdentityAssetReference:string*, AttestationIdentityKeyIsExtractable:boolean, RelyingParties[]
    com.apple.configuration.services.background-tasks
      TaskType:string*, TaskDescription:string, ExecutableAssetReference:string, LaunchdConfigurations[]
    com.apple.configuration.services.configuration-files
      ServiceType:string*, DataAssetReference:string*
    com.apple.configuration.siri.settings
      Enabled:boolean, AllowUserGeneratedContent:boolean, AllowWhileLocked:boolean, ForceProfanityFilter:boolean
    com.apple.configuration.softwareupdate.enforcement.specific
      TargetOSVersion:string*, TargetBuildVersion:string, TargetLocalDateTime:string*, DetailsURL:string
    com.apple.configuration.softwareupdate.settings
      Notifications:boolean, Deferrals{CombinedPeriodInDays:integer(1-90), MajorPeriodInDays:integer(1-90), MinorPeriodInDays:integer(1-90), SystemPeriodInDays:integer(1-90)}, RecommendedCadence:string(All|Oldest|Newest), AutomaticActions{Download:string(Allowed|AlwaysOn|AlwaysOff), InstallOSUpdates:string(Allowed|AlwaysOn|AlwaysOff), InstallSecurityUpdate:string(Allowed|AlwaysOn|AlwaysOff)}, RapidSecurityResponse{Enable:boolean, EnableRollback:boolean}, AllowStandardUserOSUpdates:boolean, Beta{ProgramEnrollment:string(Allowed|AlwaysOn|AlwaysOff), OfferPrograms[], RequireProgram{Description:string*, Token:string*}}
    com.apple.configuration.watch.enrollment
      EnrollmentProfileURL:string*, AnchorCertificateAssetReferences[]`;

    // Apple's published .mobileconfig payloads, pruned to the one thing a generator cannot recover on its own:
    // the exact key names a payload accepts, cased as Apple cases them.  Value types, ranges, per-OS availability
    // and the keys nested inside dictionaries and arrays are dropped -- this list settles what exists, not what it
    // accepts.  Regenerate by walking apple/device-management's mdm/profiles, including CommonPayloadKeys.yaml and
    // TopLevel.yaml, whose keys lead the list because they are the ones a payload manifest never repeats.
    //eslint-disable-next-line camelcase
    const MOBILECONFIG_PAYLOAD_SCHEMA_v1 = `(common keys -- valid on every dict inside PayloadContent)
      PayloadIdentifier*, PayloadUUID*, PayloadType*, PayloadVersion*, PayloadDescription, PayloadDisplayName, PayloadOrganization
    (root dict only -- never inside PayloadContent)
      PayloadContent[]*, EncryptedPayloadContent, PayloadRemovalDisallowed, PayloadScope, RemovalDate, DurationUntilRemoval,
      PayloadExpirationDate, TargetDeviceType, ConsentText{}
    com.apple.ADCertificate.managed
      CertServer*, CertTemplate*, Description, CertificateRenewalTimeInterval, CertificateAuthority, CertificateAcquisitionMechanism, AllowAllAppsAccess, PromptForCredentials, KeyIsExtractable, Keysize, EnableAutoRenewal
    com.apple.AIM.account
      AIMAccountDescription, AIMHostName*, AIMUserName, AIMPassword, AIMUseSSL, AIMPort, AIMAuthentication*
    com.apple.AssetCache.managed
      AllowCacheDelete, AllowPersonalCaching, AllowSharedCaching, AutoActivation, AutoEnableTetheredCaching, CacheLimit, DataPath, DenyTetheredCaching, DisplayAlerts, KeepAwake, ListenRanges[], ListenRangesOnly, ListenWithPeersAndParents, LocalSubnetsOnly, LogClientIdentity, Parents[], ParentSelectionPolicy, PeerFilterRanges[], PeerListenRanges[], PeerLocalSubnetsOnly, Port, PublicRanges[]
    com.apple.Dictionary
      parentalControl*
    com.apple.DirectoryService.managed
      HostName*, UserName, Password, ClientID, Description, ADOrganizationalUnit, ADMountStyle, ADCreateMobileAccountAtLoginFlag, ADCreateMobileAccountAtLogin, ADWarnUserBeforeCreatingMAFlag, ADWarnUserBeforeCreatingMA, ADForceHomeLocalFlag, ADForceHomeLocal, ADUseWindowsUNCPathFlag, ADUseWindowsUNCPath, ADAllowMultiDomainAuthFlag, ADAllowMultiDomainAuth, ADDefaultUserShellFlag, ADDefaultUserShell, ADMapUIDAttributeFlag, ADMapUIDAttribute, ADMapGIDAttributeFlag, ADMapGIDAttribute, ADMapGGIDAttributeFlag, ADMapGGIDAttribute, ADPreferredDCServerFlag, ADPreferredDCServer, ADDomainAdminGroupListFlag, ADDomainAdminGroupList[], ADNamespaceFlag, ADNamespace, ADPacketSignFlag, ADPacketSign, ADPacketEncryptFlag, ADPacketEncrypt, ADRestrictDDNSFlag, ADRestrictDDNS[], ADTrustChangePassIntervalDaysFlag, ADTrustChangePassIntervalDays
    com.apple.DiscRecording
      BurnSupport*
    com.apple.MCX
      EnableGuestAccount, DisableGuestAccount, com.apple.EnergySaver.desktop.ACPower{}, com.apple.EnergySaver.portable.ACPower{}, com.apple.EnergySaver.portable.BatteryPower{}, com.apple.EnergySaver.desktop.Schedule{}, SleepDisabled, dontAllowFDEDisable, dontAllowFDEEnable, DestroyFVKeyOnStandby, com.apple.cachedaccounts.CreateAtLogin, com.apple.cachedaccounts.WarnOnCreate, cachedaccounts.WarnOnCreate.allowNever, cachedaccounts.expiry.delete.disusedSeconds, cachedaccounts.askForSecureTokenAuthBypass, timeServer, timeZone, RequireAdminForIBSS, RequireAdminForAirPortNetworkChange, RequireAdminToTurnAirPortOnOff
    com.apple.MCX.FileVault2
      Enable*, Defer, UserEntersMissingInfo, UseRecoveryKey, ShowRecoveryKey, OutputPath, Certificate, PayloadCertificateUUID, Username, Password, UseKeychain, DeferForceAtUserLoginMaxBypassAttempts, DeferDontAskAtUserLogout, ForceEnableInSetupAssistant
    com.apple.MCX.TimeMachine
      AutoBackup, BackupAllVolumes, BackupDestURL*, BackupSizeMB, BackupSkipSys, MobileBackups, BasePaths[], SkipPaths[]
    com.apple.ManagedClient.preferences
      PayloadContent{}*
    com.apple.NSExtension
      AllowedExtensions[], DeniedExtensions[], DeniedExtensionPoints[]
    com.apple.SetupAssistant.managed
      SkipCloudSetup, SkipSiriSetup, SkipPrivacySetup, SkipiCloudStorageSetup, SkipTrueTone, SkipAppearance, SkipTouchIDSetup, SkipScreenTime, SkipAccessibility, SkipSetupItems[], SkipUnlockWithWatch, SkipWallpaper
    com.apple.ShareKitHelper
      SHKAllowedShareServices[], SHKDeniedShareServices[]
    com.apple.SoftwareUpdate
      CatalogURL, AllowPreReleaseInstallation, restrict-software-update-require-admin-to-install, AutomaticallyInstallMacOSUpdates, AutomaticallyInstallAppUpdates, AutomaticCheckEnabled, AutomaticDownload, CriticalUpdateInstall, ConfigDataInstall
    com.apple.SystemConfiguration
      Proxies{}*
    com.apple.TCC.configuration-profile-policy
      Services{}*
    com.apple.airplay.security
      SecurityType*, AccessType*, Password
    com.apple.airplay
      AllowList[], Passwords[], Whitelist[]
    com.apple.airprint
      AirPrint[]*
    com.apple.apn.managed
      DefaultsData{}*, DefaultsDomainName*
    com.apple.app.lock
      App{}*
    com.apple.applicationaccess.new
      familyControlsEnabled*, allowList[], whiteList[], pathDenyList[], pathBlackList[], pathAllowList[], pathWhiteList[]
    com.apple.applicationaccess
      allowAccountModification, allowActivityContinuation, allowAddingGameCenterFriends, allowAirDrop, allowAirPlayIncomingRequests, allowAirPrint, allowAirPrintCredentialsStorage, allowAirPrintiBeaconDiscovery, allowAppCellularDataModification, allowAppClips, allowAppInstallation, allowAppleIntelligenceReport, allowApplePersonalizedAdvertising, allowAppRemoval, allowAppsToBeHidden, allowAppsToBeLocked, allowARDRemoteManagementModification, allowAssistant, allowAssistantUserGeneratedContent, allowAssistantWhileLocked, allowAutoCorrection, allowAutoDim, allowAutomaticAppDownloads, allowAutomaticScreenSaver, allowAutoUnlock, allowBluetoothModification, allowBluetoothSharingModification, allowBookstore, allowBookstoreErotica, allowCallRecording, allowCamera, allowCellularPlanModification, allowChat, allowCloudAddressBook, allowCloudBackup, allowCloudBookmarks, allowCloudCalendar, allowCloudDesktopAndDocuments, allowCloudDocumentSync, allowCloudFreeform, allowCloudKeychainSync, allowCloudMail, allowCloudNotes, allowCloudPhotoLibrary, allowCloudPrivateRelay, allowCloudReminders, allowContentCaching, allowContinuousPathKeyboard, allowDefaultBrowserModification, allowDefaultCallingAppModification, allowDefaultMessagingAppModification, allowDefinitionLookup, allowDeviceNameModification, allowDeviceSleep, allowDiagnosticSubmission, allowDiagnosticSubmissionModification, allowDictation, allowedCameraRestrictionBundleIDs[], allowedExternalIntelligenceWorkspaceIDs[], allowEnablingRestrictions, allowEnterpriseAppTrust, allowEnterpriseBookBackup, allowEnterpriseBookMetadataSync, allowEraseContentAndSettings, allowESIMModification, allowESIMOutgoingTransfers, allowExplicitContent, allowExternalIntelligenceIntegrations, allowExternalIntelligenceIntegrationsSignIn, allowFileSharingModification, allowFilesNetworkDriveAccess, allowFilesUSBDriveAccess, allowFindMyDevice, allowFindMyFriends, allowFindMyFriendsModification, allowFingerprintForUnlock, allowFingerprintModification, allowGameCenter, allowGenmoji, allowGlobalBackgroundFetchWhenRoaming, allowHostPairing, allowImagePlayground, allowImageWand, allowInAppPurchases, allowInternetSharingModification, allowiPhoneMirroring, allowiPhoneWidgetsOnMac, allowiTunes, allowiTunesFileSharing, allowKeyboardShortcuts, allowListedAppBundleIDs[], allowLiveVoicemail, allowLocalUserCreation, allowLockScreenControlCenter, allowLockScreenNotificationsView, allowLockScreenTodayView, allowMailPrivacyProtection, allowMailSmartReplies, allowMailSummary, allowManagedAppsCloudSync, allowManagedToWriteUnmanagedContacts, allowMarketplaceAppInstallation, allowMediaSharingModification, allowMultiplayerGaming, allowMusicService, allowNews, allowNFC, allowNotesTranscription, allowNotesTranscriptionSummary, allowNotificationsModification, allowOpenFromManagedToUnmanaged, allowOpenFromUnmanagedToManaged, allowOTAPKIUpdates, allowPairedWatch, allowPassbookWhileLocked, allowPasscodeModification, allowPasswordAutoFill, allowPasswordProximityRequests, allowPasswordSharing, allowPersonalHotspotModification, allowPersonalizedHandwritingResults, allowPhotoStream, allowPodcasts, allowPredictiveKeyboard, allowPrinterSharingModification, allowProximitySetupToNewDevice, allowRadioService, allowRapidSecurityResponseInstallation, allowRapidSecurityResponseRemoval, allowRCSMessaging, allowRemoteAppleEventsModification, allowRemoteAppPairing, allowRemoteScreenObservation, allowRosettaUsageAwareness, allowSafari, allowSafariHistoryClearing, allowSafariPrivateBrowsing, allowSafariSummary, allowSatelliteConnection, allowScreenShot, allowSharedDeviceTemporarySession, allowSharedStream, allowSiriAI, allowSpellCheck, allowSpotlightInternetResults, allowStartupDiskModification, allowSystemAppRemoval, allowTimeMachineBackup, allowUIAppInstallation, allowUIConfigurationProfileInstallation, allowUniversalControl, allowUnmanagedToReadManagedContacts, allowUnpairedExternalBootToRecovery, allowUntrustedTLSPrompt, allowUSBRestrictedMode, allowVideoConferencing, allowVideoConferencingRemoteControl, allowVisualIntelligenceSummary, allowVoiceDialing, allowVPNCreation, allowWallpaperModification, allowWebDistributionAppInstallation, allowWritingTools, autonomousSingleAppModePermittedAppIDs[], blacklistedAppBundleIDs[], blockedAppBundleIDs[], deniedICCIDsForiMessageFaceTime[], deniedICCIDsForRCS[], enforcedFingerprintTimeout, enforcedSoftwareUpdateDelay, enforcedSoftwareUpdateMajorOSDeferredInstallDelay, enforcedSoftwareUpdateMinorOSDeferredInstallDelay, enforcedSoftwareUpdateNonOSDeferredInstallDelay, forceAirDropUnmanaged, forceAirPlayIncomingRequestsPairingPassword, forceAirPlayOutgoingRequestsPairingPassword, forceAirPrintTrustedTLSRequirement, forceAssistantProfanityFilter, forceAuthenticationBeforeAutoFill, forceAutomaticDateAndTime, forceBypassScreenCaptureAlert, forceClassroomAutomaticallyJoinClasses, forceClassroomRequestPermissionToLeaveClasses, forceClassroomUnpromptedAppAndDeviceLock, forceClassroomUnpromptedScreenObservation, forceDelayedAppSoftwareUpdates, forceDelayedMajorSoftwareUpdates, forceDelayedSoftwareUpdates, forceEncryptedBackup, forceITunesStorePasswordEntry, forceLimitAdTracking, forceOnDeviceOnlyDictation, forceOnDeviceOnlyTranslation, forcePreserveESIMOnErase, forceWatchWristDetection, forceWiFiPowerOn, forceWiFiToAllowedNetworksOnly, forceWiFiWhitelisting, ratingApps, ratingAppsExemptedBundleIDs[], ratingMovies, ratingRegion, ratingTVShows, requireManagedPasteboard, safariAcceptCookies, safariAllowAutoFill, safariAllowJavaScript, safariAllowPopups, safariForceFraudWarning, whitelistedAppBundleIDs[]
    com.apple.appstore
      restrict-store-require-admin-to-install, restrict-store-softwareupdate-only, restrict-store-disable-app-adoption, DisableSoftwareUpdateNotifications
    com.apple.asam
      AllowedApplications[]*
    com.apple.associated-domains
      Configuration[]*
    com.apple.caldav.account
      CalDAVAccountDescription, CalDAVHostName*, CalDAVUsername, CalDAVPassword, CalDAVPrincipalURL, CalDAVUseSSL, CalDAVPort, VPNUUID
    com.apple.carddav.account
      CardDAVAccountDescription, CardDAVHostName*, CardDAVUsername, CardDAVPassword, CardDAVPrincipalURL, CardDAVUseSSL, CardDAVPort, CommunicationServiceRules{}, VPNUUID
    com.apple.cellular
      AttachAPN{}, APNs[]
    com.apple.cellularprivatenetwork.managed
      Geofences[], DataSetName*, VersionNumber*, CellularDataPreferred, EnableNRStandalone, NetworkIdentifier, CsgNetworkIdentifier
    com.apple.conferenceroomdisplay
      Message
    com.apple.configurationprofile.identification
      PayloadIdentification{}*
    com.apple.dashboard
      whiteListEnabled*, WhiteList[]*
    com.apple.declarations
      Declarations[]*
    com.apple.desktop
      locked, override-picture-path
    com.apple.dnsProxy.managed
      AppBundleIdentifier*, ProviderBundleIdentifier, ProviderDesignatedRequirement, ProviderConfiguration{}, DNSProxyUUID
    com.apple.dnsSettings.managed
      DNSSettings{}*, OnDemandRules[], ProhibitDisablement
    com.apple.dock
      tilesize, size-immutable, magnification, magnify-immutable, largesize, magsize-immutable, orientation, position-immutable, mineffect, mineffect-immutable, windowtabbing, windowtabbing-immutable, dblclickbehavior, dblclickbehavior-immutable, minimize-to-application, minintoapp-immutable, launchanim, launchanim-immutable, autohide, autohide-immutable, show-process-indicators, showindicators-immutable, show-recents, showrecents-immutable, contents-immutable, MCXDockSpecialFolders[], AllowDockFixupOverride, static-only, static-others[], static-apps[], persistent-apps[], persistent-others[]
    com.apple.domains
      EmailDomains[], WebDomains[], SafariPasswordAutoFillDomains[], CrossSiteTrackingPreventionRelaxedDomains[], CrossSiteTrackingPreventionRelaxedApps[]
    com.apple.eas.account
      EmailAddress, Host, SSL, OAuth, UserName, Password, Certificate, CertificateName, CertificatePassword, PreventMove, PreventAppSheet, PayloadCertificateUUID, SMIMEEnabled, SMIMESigningEnabled, SMIMESigningCertificateUUID, SMIMEEncryptionEnabled, SMIMEEncryptionCertificateUUID, SMIMEEnablePerMessageSwitch, disableMailRecentsSyncing, MailNumberOfPastDaysToSync, HeaderMagic, CommunicationServiceRules{}, allowMailDrop, SMIMESigningUserOverrideable, SMIMESigningCertificateUUIDUserOverrideable, SMIMEEncryptByDefault, SMIMEEncryptByDefaultUserOverrideable, SMIMEEncryptionCertificateUUIDUserOverrideable, SMIMEEnableEncryptionPerMessageSwitch, EnableMail, EnableContacts, EnableCalendars, EnableReminders, EnableNotes, EnableMailUserOverridable, EnableContactsUserOverridable, EnableCalendarsUserOverridable, EnableRemindersUserOverridable, EnableNotesUserOverridable, OAuthSignInURL, OAuthTokenRequestURL, OverridePreviousPassword, VPNUUID
    com.apple.education
      OrganizationUUID*, OrganizationName*, PayloadCertificateUUID, LeaderPayloadCertificateAnchorUUID[], MemberPayloadCertificateAnchorUUID[], ResourcePayloadCertificateUUID, UserIdentifier*, Departments[], Groups[]*, Users[]*, DeviceGroups[], ScreenObservationPermissionModificationAllowed
    com.apple.ews.account
      EmailAddress, Host, SSL, OAuth, OAuthSignInURL, UserName, Password, PayloadCertificateUUID, AuthenticationCertificateUUID, allowMailDrop, Path, Port, ExternalHost, ExternalSSL, ExternalPath, ExternalPort
    com.apple.extensiblesso
      ExtensionIdentifier*, TeamIdentifier*, Type*, Realm*, ExtensionData{}, Hosts[], TeamIdentifier, Realm, URLs[], ScreenLockedBehavior, DeniedBundleIdentifiers[], AuthenticationMethod, RegistrationToken, PlatformSSO{}
    com.apple.familycontrols.contentfilter
      restrictWeb*, useContentFilter, allowListEnabled, whitelistEnabled, siteAllowList[], siteWhitelist[], filterAllowList[], filterWhitelist[], filterDenyList[], filterBlacklist[]
    com.apple.familycontrols.timelimits.v2
      familyControlsEnabled*, time-limits{}
    com.apple.fileproviderd
      ManagementAllowsRemoteSyncing, ManagementRemoteSyncingAllowList[], AllowManagedFileProvidersToRequestAttribution, ManagementAllowsKnownFolderSyncing, ManagementKnownFolderSyncingAllowList[], ManagementAllowsExternalVolumeSyncing, ManagementExternalVolumeSyncingAllowList[], ManagementDomainAutoEnablementList[]
    com.apple.finder
      ProhibitBurn, ProhibitConnectTo, ProhibitEject, ProhibitGoToFolder, ShowExternalHardDrivesOnDesktop, ShowHardDrivesOnDesktop, ShowMountedServersOnDesktop, ShowRemovableMediaOnDesktop, WarnOnEmptyTrash
    com.apple.firstactiveethernet.managed
      ANY
    com.apple.firstethernet.managed
      ANY
    com.apple.font
      Name, Font*
    com.apple.gamed
      GKFeatureGameCenterAllowed, GKFeatureAccountModificationAllowed, GKFeatureAddingGameCenterFriendsAllowed, GKFeatureMultiplayerGamingAllowed
    com.apple.globalethernet.managed
      ANY
    com.apple.google-oauth
      AccountDescription, AccountName, EmailAddress*, CommunicationServiceRules{}, VPNUUID
    com.apple.homescreenlayout
      Dock[], Pages[]*
    com.apple.ironwood.support
      Profanity Allowed, Ironwood Allowed
    com.apple.jabber.account
      JabberAccountDescription, JabberHostName*, JabberUserName, JabberPassword, JabberUseSSL, JabberPort, JabberAuthentication*
    com.apple.ldap.account
      LDAPAccountDescription, LDAPAccountHostName*, LDAPAccountUserName, LDAPAccountPassword, LDAPAccountUseSSL, LDAPSearchSettings[], VPNUUID
    com.apple.loginitems.managed
      AutoLaunchedApplicationDictionary-managed[]*
    com.apple.loginwindow
      SHOWFULLNAME, HideLocalUsers, IncludeNetworkUser, HideAdminUsers, SHOWOTHERUSERS_MANAGED, AdminHostInfo, AdminMayDisableMCX, AllowList[], DenyList[], HideMobileAccounts, ShutDownDisabled, RestartDisabled, RetriesUntilHint, SleepDisabled, DisableConsoleAccess, LoginwindowText, ShutDownDisabledWhileLoggedIn, RestartDisabledWhileLoggedIn, PowerOffDisabledWhileLoggedIn, LogOutDisabledWhileLoggedIn, DisableScreenLockImmediate, showInputMenu, DisableFDEAutoLogin, AutologinUsername, AutologinPassword, ForceWifiConfigurationOnLockScreen, ForceCaptivePortalConnectionFromLockScreen
    com.apple.lom
      DeviceCertificateUUID, ControllerCertificateUUID, DeviceCACertificateUUIDs[], ControllerCACertificateUUIDs[]
    com.apple.mail.managed
      EmailAccountDescription, EmailAccountName, EmailAccountType*, EmailAddress, IncomingMailServerAuthentication*, IncomingMailServerHostName*, IncomingMailServerPortNumber, IncomingMailServerUseSSL, IncomingMailServerUsername, IncomingPassword, OutgoingPassword, OutgoingPasswordSameAsIncomingPassword, OutgoingMailServerAuthentication*, OutgoingMailServerHostName*, OutgoingMailServerPortNumber, OutgoingMailServerUseSSL, OutgoingMailServerUsername, PreventMove, PreventAppSheet, SMIMEEnabled, SMIMESigningEnabled, SMIMESigningCertificateUUID, SMIMEEncryptionEnabled, SMIMEEncryptionCertificateUUID, SMIMEEnablePerMessageSwitch, disableMailRecentsSyncing, allowMailDrop, IncomingMailServerIMAPPathPrefix, SMIMESigningUserOverrideable, SMIMESigningCertificateUUIDUserOverrideable, SMIMEEncryptByDefault, SMIMEEncryptByDefaultUserOverrideable, SMIMEEncryptionCertificateUUIDUserOverrideable, SMIMEEnableEncryptionPerMessageSwitch, VPNUUID
    com.apple.mcxMenuExtras
      delaySeconds, maxWaitSeconds, AirPort.menu, Battery.menu, Bluetooth.menu, CPU.menu, Clock.menu, Displays.menu, Eject.menu, Fax.menu, HomeSync.menu, iChat.menu, Ink.menu, IrDA.menu, PCCard.menu, PPP.menu, PPPoE.menu, RemoteDesktop.menu, Script Menu.menu, Spaces.menu, Sync.menu, TextInput.menu, TimeMachine.menu, UniversalAccess.menu, User.menu, VPN.menu, Volume.menu, WWAN.menu
    com.apple.mcxloginscripts
      loginscripts[], logoutscripts[], skipLoginHook, skipLogoutHook
    com.apple.mcxprinting
      RequireAdminToAddPrinters, AllowLocalPrinters, RequireAdminToPrintLocally, ShowOnlyManagedPrinters, PrintFooter, PrintMACAddress, FooterFontSize, FooterFontName, DefaultPrinter{}, UserPrinterList{}
    com.apple.mdm
      IdentityCertificateUUID*, Topic*, ServerURL*, CheckInURL, SignMessage, AccessRights, UseDevelopmentAPNS, ManagedAppleID, AssignedManagedAppleID, EnrollmentMode, ServerURLPinningCertificateUUIDs[], CheckInURLPinningCertificateUUIDs[], PinningRevocationCheckRequired, ServerCapabilities[], CheckOutWhenRemoved, RequiredAppIDForMDM, PromptUserToAllowBootstrapTokenForAuthentication
    com.apple.mobiledevice.passwordpolicy
      allowSimple, forcePIN, maxFailedAttempts, maxInactivity, maxPINAgeInDays, minComplexChars, minLength, requireAlphanumeric, pinHistory, maxGracePeriod, minutesUntilFailedLoginReset, changeAtNextAuth, customRegex{}
    com.apple.networkusagerules
      ApplicationRules[], SIMRules[]
    com.apple.notificationsettings
      NotificationSettings[]*
    com.apple.osxserver.account
      HostName*, UserName*, Password, AccountDescription, ConfiguredAccounts[]*
    com.apple.preference.security
      dontAllowPasswordResetUI, dontAllowLockMessageUI, dontAllowFireWallUI
    com.apple.preference.users
      DisableUsingiCloudPassword
    com.apple.profileRemovalPassword
      RemovalPassword
    com.apple.proxy.http.global
      ProxyType, ProxyServer, ProxyServerPort, ProxyUsername, ProxyPassword, ProxyPACURL, ProxyPACFallbackAllowed, ProxyCaptiveLoginAllowed
    com.apple.relay.managed
      Relays[]*, MatchDomains[], ExcludedDomains[], MatchFQDNs[], ExcludedFQDNs[], RelayUUID, UIToggleEnabled, AllowDNSFailover
    com.apple.screensaver.user
      moduleName*, modulePath, idleTime
    com.apple.screensaver
      askForPassword, askForPasswordDelay, idleTime, loginWindowModulePath, moduleName*
    com.apple.secondactiveethernet.managed
      ANY
    com.apple.secondethernet.managed
      ANY
    com.apple.security.FDERecoveryKeyEscrow
      Location*, EncryptCertPayloadUUID*, DeviceKey
    com.apple.security.FDERecoveryRedirect
      RedirectURL*, EncryptCertPayloadUUID*
    com.apple.security.acme
      DirectoryURL*, ClientIdentifier*, KeySize*, KeyType*, HardwareBound*, Subject[]*, SubjectAltName{}, UsageFlags, ExtendedKeyUsage[], Attest, KeyIsExtractable, AllowAllAppsAccess
    com.apple.security.certificatepreference
      Name*, PayloadCertificateUUID*
    com.apple.security.certificaterevocation
      EnabledForCerts[]
    com.apple.security.certificatetransparency
      DisabledForCerts[], DisabledForDomains[]
    com.apple.security.firewall
      EnableFirewall*, BlockAllIncoming, EnableStealthMode, Applications[], AllowSigned, AllowSignedApp
    com.apple.security.identitypreference
      Name*, PayloadCertificateUUID*
    com.apple.security.pem
      PayloadCertificateFileName, PayloadContent*
    com.apple.security.pkcs1
      PayloadCertificateFileName, PayloadContent*
    com.apple.security.pkcs12
      PayloadCertificateFileName, PayloadContent*, Password, AllowAllAppsAccess, KeyIsExtractable
    com.apple.security.root
      PayloadCertificateFileName, PayloadContent*
    com.apple.security.scep
      PayloadContent{}*
    com.apple.security.smartcard
      UserPairing, allowSmartCard, checkCertificateTrust, oneCardPerUser, tokenRemovalAction, enforceSmartCard
    com.apple.servicemanagement
      Rules[]*
    com.apple.shareddeviceconfiguration
      AssetTagInformation, IfLostReturnToMessage, LockScreenFootnote
    com.apple.sso
      Name*, Kerberos{}
    com.apple.subscribedcalendar.account
      SubCalAccountDescription, SubCalAccountHostName*, SubCalAccountUsername, SubCalAccountPassword, SubCalAccountUseSSL, VPNUUID
    com.apple.syspolicy.kernel-extension-policy
      AllowNonAdminUserApprovals, AllowUserOverrides, AllowedTeamIdentifiers[], AllowedKernelExtensions{}
    com.apple.system-extension-policy
      AllowUserOverrides, AllowedTeamIdentifiers[], AllowedSystemExtensionTypes{}, AllowedSystemExtensions{}, RemovableSystemExtensions{}, NonRemovableSystemExtensions{}, NonRemovableFromUISystemExtensions{}
    com.apple.system.logging
      Subsystems{}, System{}
    com.apple.systemmigration
      CustomBehavior[]
    com.apple.systempolicy.control
      EnableAssessment, AllowIdentifiedDevelopers, EnableXProtectMalwareUpload
    com.apple.systempolicy.managed
      DisableOverride
    com.apple.systempolicy.rule
      Requirement, Comment, Priority, Expiration, OperationType, LeafCertificate
    com.apple.systempreferences
      EnabledPreferencePanes[], DisabledPreferencePanes[], DisabledSystemSettings[]
    com.apple.systemuiserver
      logout-eject{}, mount-controls{}, unmount-controls{}
    com.apple.thirdactiveethernet.managed
      ANY
    com.apple.thirdethernet.managed
      ANY
    com.apple.tvremote
      AllowedRemotes[], AllowedTVs[]
    com.apple.universalaccess
      closeViewFarPoint, closeViewHotkeysEnabled, closeViewNearPoint, closeViewScrollWheelToggle, closeViewShowPreview, closeViewSmoothImages, contrast, flashScreen, grayscale, mouseDriver, mouseDriverCursorSize, mouseDriverIgnoreTrackpad, mouseDriverInitialDelay, mouseDriverMaxSpeed, slowKey, slowKeyBeepOn, slowKeyDelay, stereoAsMono, stickyKey, stickyKeyBeepOnModifier, stickyKeyShowWindow, voiceOverOnOffKey, whiteOnBlack
    com.apple.vpn.managed.applayer
      VPNUUID*, CellularSliceUUID, SafariDomains[], MailDomains[], CalendarDomains[], ContactsDomains[], AssociatedDomains[], ExcludedDomains[], OnDemandMatchAppEnabled, SMBDomains[]
    com.apple.vpn.managed.appmapping
      AppLayerVPNMapping[]*
    com.apple.vpn.managed
      VPNType*, VPNSubType, UserDefinedName*, VendorConfig{}, VPN{}, IPv4{}, PPP{}, IPSec{}, IKEv2{}, DNS{}, Proxies{}, AlwaysOn{}, TransparentProxy{}
    com.apple.webClip.managed
      Precomposed, FullScreen, URL*, Icon, IsRemovable, Label*, IgnoreManifestScope, TargetApplicationBundleIdentifier
    com.apple.webcontent-filter
      FilterType, SafariHistoryRetentionEnabled, AutoFilterEnabled, PermittedURLs[], BlacklistedURLs[], DenyListURLs[], HideDenyListURLs, WhitelistedBookmarks[], AllowListBookmarks[], UserDefinedName, PluginBundleID, ServerAddress, UserName, Password, PayloadCertificateUUID, Organization, VendorConfig{}, FilterBrowsers, FilterSockets, FilterDataProviderDesignatedRequirement, FilterDataProviderBundleIdentifier, FilterPackets, FilterPacketProviderDesignatedRequirement, FilterPacketProviderBundleIdentifier, FilterGrade, ContentFilterUUID, FilterURLs, URLFilterParameters{}
    com.apple.wifi.managed
      AutoJoin, SSID_STR, HIDDEN_NETWORK, ProxyType, EncryptionType, Password, PayloadCertificateUUID, EAPClientConfiguration{}, DisplayedOperatorName, DomainName, RoamingConsortiumOIs[], ServiceProviderRoamingEnabled, IsHotspot, HESSID, NAIRealmNames[], MCCAndMNCs[], CaptiveBypass, QoSMarkingPolicy{}, SetupModes[], EnableIPv6, TLSCertificateRequired, ProxyServer, ProxyServerPort, ProxyUsername, ProxyPassword, ProxyPACURL, ProxyPACFallbackAllowed, DisableAssociationMACRandomization, AllowJoinBeforeFirstUnlock
    com.apple.xsan.preferences
      onlyMount[], denyMount[], denyDLC[], preferDLC[], useDLC
    com.apple.xsan
      sanName*, sanConfigURLs[], fsnameservers[], sanAuthMethod, sharedSecret*
    loginwindow
      DisableLoginItemsSuppression`;


    // The tail of the system prompt.
    let RESPONSE_SHAPE;
    if(!useLighterResponseShape) {

      RESPONSE_SHAPE = `Respond in JSON with this data shape:
      {
        "configurationProfile": "TODO",
        "profileFilename": "TODO",
        // Things the admin must do or decide that are not visible in the profile itself.
        // Empty string when there is nothing exceptional, which is the common case.
        "deliveryNotes": "",
        "settingsEnforced": [// For each setting enforced by the configuration profile.
          {
            // The name (key) of the setting that is enforced. e.g., LoginwindowText
            name: "TODO",
            // The value of the setting that is enforced
            value: "TODO",
            // Where this setting comes from: the CSP node path, the Apple payload domain and key, or the declaration type.
            schemaReference: "TODO",
            // The documented range, enum, or type this setting accepts, including the declared format.
            allowedValues: "TODO",
            // What the value above actually does, in words. e.g., "0 = a password is required"
            valueMeaning: "TODO",
            // The Apple or Microsoft reference page for this setting.
            documentationUrl: "TODO",
            // Applicability, dependencies, and any condition under which this setting deploys but does nothing. Empty string if there are none.
            caveats: "TODO"
          },
          {...}
        ]
      }

      If a configuration profile cannot be generated from the provided instructions, respond with this shape instead:
      {
        "couldNotGenerateProfile": true,
        // Explain why a profile could not be generated, naming the specific setting, node, or key that could not be confirmed. The tone should be informational and brief.
        "reasonWhyAProfileCouldNotBeGenerated": TODO
      }
      `;
    } else {
      RESPONSE_SHAPE = `Respond in JSON with this data shape:
      {
        "configurationProfile": "TODO",
        "profileFilename": "TODO",
        // Things the admin must do or decide that are not visible in the profile itself.
        // Empty string when there is nothing exceptional, which is the common case.
        "deliveryNotes": "",
        "settingsEnforced": [// For each setting enforced by the configuration profile.
          {
            name: "TODO",
            value: "TODO",
          },
          {...}
        ]
      }

      If a configuration profile cannot be generated from the provided instructions, respond with this shape instead:
      {
        "couldNotGenerateProfile": true,
        // Explain why a profile could not be generated, naming the specific setting, node, or key that could not be confirmed. The tone should be informational and brief.
        "reasonWhyAProfileCouldNotBeGenerated": TODO
      }
      `;
    }

    // Rules that apply to every profile type.  Ordered by consequence: a violation of an early rule produces a profile that deploys cleanly and does nothing.
    let sharedRules = [
      'Generate a profile that any MDM can deliver.  Use only syntax defined by Apple, Microsoft, or Google -- never a vendor-specific variable, placeholder, or extension, and never Fleet-specific syntax such as $FLEET_SECRET_ or FLEET_VAR_.  A vendor placeholder the delivering MDM does not recognize is shipped to the device as a literal value.',
      'Reproduce user-supplied identifiers character for character, including case: SSIDs, profile names, certificate subjects, domain names.  Never re-capitalize, trim, or reword them.  An SSID differing by one letter\'s case deploys cleanly and matches nothing.',
      'Use only settings you can attribute to a specific published source (an Apple payload key, a Windows CSP node, or an Apple declaration type).  If the instructions cannot be satisfied that way, do not approximate -- return the "couldNotGenerateProfile" shape instead.',
      'You have no network access and cannot open any URL.  Never state or imply that you validated this profile against a reference, a schema, or a linter.  "documentationUrl" is where a human can check your work, not evidence that you checked it.',
      'Enforce only what the instructions ask for.  The only settings you may add beyond the request are ones the requested setting depends on, and each of those must be called out in "caveats".',
      'Write credentials the admin supplied as literals, since the profile is unusable without them.  Do not invent a placeholder.  Note in "deliveryNotes" that the file contains a cleartext credential.',
      'When a platform requires a companion artifact the profile cannot contain -- a DDM activation declaration, a referenced asset declaration -- generate the configuration itself and describe the companion in "deliveryNotes".',
      'If a setting is commonly managed by an MDM directly rather than by a custom profile, such as disk encryption, still generate the profile as asked and note in "deliveryNotes" that some MDMs manage this natively and may reject or conflict with a custom profile.',
      'Escape newlines inside "configurationProfile" as \\n so the surrounding JSON stays valid.  Do not emit raw line breaks inside the string, and do not collapse the profile onto one line.',
    ];

    // "deliveryNotes" defaults to noise unless it is aggressively constrained.  An empty string is a weaker affordance than an empty array, so these rules carry more of the load.
    let deliveryNotesRules = [
      '"deliveryNotes" is for exceptions only: something the admin has to do or decide that is not visible in the profile itself.  Use an empty string when nothing applies.  An empty string is the right answer for most profiles -- prefer it whenever you are unsure whether a note earns its place.',
      'Never write a sentence stating that a condition does not apply.  "No credentials or secrets are embedded" and "no companion declaration is required" are not notes -- leaving them out already says that.',
      'Never restate what the profile is, which platform it targets, or how that platform is normally delivered.  The admin chose the format and already knows.',
      'One sentence per action, addressed to the admin, and no more than two sentences in total.  For example: "Replace the passphrase with a secret variable before committing this to a repository."',
    ];

    let promptConfigByProfileType = {

      'csp': {
        description: 'CSP XML profile that enforces OS settings on Windows devices',
        // How the triage prompt describes a setting the references below actually cover.  Kept
        // next to those references so the two cannot drift apart.
        firstPartySettingDescription: 'a node in a Microsoft-published CSP',
        references: [
          'Windows CSP nodes, formats, and allowed values: https://learn.microsoft.com/en-us/windows/client-management/mdm/',
        ],
        rules: [
          // Document shape.  Wrong here and Fleet rejects the file on upload.
          'A Windows profile is a sequence of OMA-DM command elements, not a SyncML document.  The top level must be one or more <Add>, <Replace>, <Exec>, or <Atomic> elements and nothing else.  Never wrap them in <SyncML>, <SyncBody>, <Final/>, or an invented container such as <WindowsCSP>, and never emit an XML declaration -- the SyncML envelope carries session state (SyncHdr, SessionID, MsgID) that only the MDM server can populate at delivery time.',
          'If you use <Atomic>, it must wrap every command in the profile.  Never mix an <Atomic> element with sibling top-level commands, and never nest one <Atomic> inside another.',
          // Node identity.
          'Emit only LocURI paths that exist in a published CSP.  Never construct, guess, or extrapolate a path, and name the CSP you used in "schemaReference".',
          'Percent-encode any character in a LocURI path segment that is not URI-legal.  An SSID named "Cool Network" is Cool%20Network in the LocURI, while the value inside the payload keeps its literal spaces.',
          // Type integrity.  The largest source of silent failure.
          'In the Policy CSP, settings that read as boolean are almost always DFFormat int with allowed values 0 and 1, not bool.  Default to <Format>int</Format> with numeric <Data>.  Use bool only when the node genuinely declares it, and say so in "caveats".',
          'Before returning, re-read every Item and confirm <Data> is legal for the <Format> you declared: int accepts digits only, bool accepts literally true or false, chr accepts text.  If you have written 0 or 1 as the data, the format is int, not bool.',
          'Never infer a boolean value from the node name.  DeviceLock/DevicePasswordEnabled takes 0 for enabled and 1 for disabled.  Always state what the value you chose actually does in "valueMeaning".',
          'State the node\'s declared DFFormat verbatim in "allowedValues", next to the format you emitted, so a reviewer can compare the two side by side.',
          'If you cannot recall a node\'s documented DFFormat with confidence, do not guess -- return the "couldNotGenerateProfile" shape and name the node you were unsure about.',
          'ADMX-backed Policy CSP nodes are never int.  They declare DFFormat chr and take a policy fragment as their value: <Format>chr</Format> with <Data><![CDATA[<enabled/>]]></Data> or <![CDATA[<disabled/>]]>, plus a nested <data id="..." value="..."/> element for each sub-option the admin actually asked for and none for the ones they did not.  Writing 1 or 0 to one of these nodes deploys cleanly and enforces nothing.',
          'Never derive the area segment of a Policy CSP LocURI from the name of the .admx file that backs it.  Most ADMX-backed areas are named ADMX_<AdmxFileName>, but plenty are not: PowerShell script block logging lives under WindowsPowerShell, not ADMX_PowerShellExecutionPolicy or ADMX_PowerShell, even though PowerShellExecutionPolicy.admx is the backing file.  Use the area exactly as the published node lists it, and if you cannot recall it, return the "couldNotGenerateProfile" shape rather than constructing one that looks right.',
          // Verb and dependencies.
          'Use Replace for Policy CSP leaf nodes that hold a value.  Reserve Add for nodes that do not exist until you create them, such as ADMXInstall, certificate installs, and WiFi or VPN profile instances.  When a node\'s AccessType permits both, choose Replace, because the profile may reach a host where the value is already set and Add can fail there.',
          'Only use an Add or Replace verb on a node whose AccessType permits it.  Roughly 500 leaf nodes accept only Get, and a Replace against one is accepted, deploys, and then fails on the device.',
          'Include the nodes the requested setting depends on.  DeviceLock/MinDevicePasswordLength has no effect unless DeviceLock/DevicePasswordEnabled is also set.',
          'Record the minimum OS build and the Windows editions each node applies to in "caveats".  A valid setting aimed at the wrong SKU is a silent no-op, not an error.',
          // Embedded values.
          'When a node\'s value is embedded XML -- WiFi/Profile/*/WlanXml, ADMXInstall, and similar -- wrap it in <![CDATA[ ... ]]> rather than escaping it as entities, and emit that value as a single line with no line breaks or indentation between its elements.  This applies only to the embedded value; the surrounding SyncML keeps its normal indentation.',
          'Be consistent within a profile about optional <Meta> children.  If you emit <Type> for one item, emit it for all of them, and namespace every <Meta> child as xmlns="syncml:metinf".',
          'For WiFi profiles, emit both <name> and <hex> inside <SSID>, where <hex> is the uppercase hex encoding of the SSID bytes.  Windows and some MDMs treat the hex form as authoritative when both are present.',
        ],
      },

      'mobileconfig': {
        description: 'XML .mobileconfig profile that enforces OS settings on macOS/iOS/ipadOS devices',
        firstPartySettingDescription: 'a key in an Apple-published payload',
        providedSchema: MOBILECONFIG_PAYLOAD_SCHEMA_v1,//eslint-disable-line camelcase
        providedSchemaDescription: `Provided context: every payload type Apple publishes a manifest for, and the top-level keys each one accepts.
Format is a payload type followed by its keys, where \`*\` marks a required key, \`{}\` is a dictionary, and \`[]\` is
an array.  The two parenthesized blocks that open the list are not payload types: the first gives the keys every dict
inside PayloadContent may carry, and the second the keys that belong on the root dict and nowhere else.

This list is names only.  It settles how a listed payload type's keys are spelled and cased, and for a listed payload
type it is complete: a top-level key not listed under it does not exist in it.  It does not give value types, allowed
values, or the keys nested inside a \`{}\` or \`[]\`, so recall those from the payload's documentation as usual, or
return the "couldNotGenerateProfile" shape.

Absence from this list is not proof a domain is unusable.  Apple preference domains that have no MDM payload manifest
are managed as preference domains and are not listed here, and neither are third-party domains, which come from the ProfileManifests reference.
What absence does rule out is a first-party payload type of your own invention.
When a listed payload type covers the setting, use it, and fall back to an unlisted preference domain only when nothing listed does.`,
        references: [
          'First-party Apple payloads: the payload types and their top-level keys are provided below -- https://github.com/apple/device-management/tree/release/mdm/profiles is where a human can check them.',
          'Third-party Apple payloads: https://github.com/ProfileManifests/ProfileManifests',
        ],
        rules: [
          // Third-party payloads.
          'If this is an attempt to change a third-party application\'s settings, use that application\'s preference domain -- com.google.Chrome, us.zoom.config and its keys must come from the ProfileManifests reference.',
          // Document shape.
          'Emit valid property list XML: the plist DOCTYPE, plist version="1.0", and correctly typed values.',
          'Include PayloadIdentifier, PayloadType, PayloadUUID, PayloadVersion, and PayloadDisplayName on the root dict and on every dict inside PayloadContent.  The root PayloadType is "Configuration" and PayloadVersion is 1.',
          'Keep the plist indented and readable across multiple lines.  Do not collapse it onto one line.',
          // Key fidelity.
          'Apple payload keys are not consistently cased, and the inconsistency is inside a single dict.  The passcode payload uses forcePIN, minLength, and allowSimple -- lowercase first letter -- beside PascalCase PayloadIdentifier and PayloadType.  Reproduce every key exactly as documented for its payload type.  Never normalize casing in either direction.',
          'Use only keys documented for the payload type you chose.  An invented key is written into the profile and nothing downstream rejects it, so the profile looks right and does nothing.',
          'When more than one key in a payload could plausibly satisfy the request, choose by documented meaning, say what the chosen value actually does in valueMeaning, and return couldNotGenerateProfile rather than guessing between them.',
          'When the payload type you need appears in the provided list, copy it and its top-level keys character for character, casing included, and use no top-level key the list does not give it.  A key you remember differently than the list spells it is the list\'s spelling, not yours.',
          'The provided list stops at the top level, and it does not cover preference domains that have no payload manifest.  If you cannot recall the keys inside a dictionary or array member, the keys of an unlisted preference domain, or the value type or allowed values of any key, do not guess and do not adapt a key from a DDM declaration -- return the "couldNotGenerateProfile" shape and name the payload type you were unsure about.',
          // Value typing.
          'Type every value as plist: <true/> or <false/> for booleans, never <string>true</string>; <integer> for whole numbers; <real> for decimals; <data> with base64 for binary; <date> with an ISO 8601 timestamp.',
          // Dependencies
          'Include the keys the requested setting depends on. minLength and allowSimple enforce nothing unless forcePIN is also set.',
          'Beyond the keys the request names and the keys those depend on, a key\'s presence in the provided list is not a reason to set it.',
          // Identifiers.
          'Take every PayloadUUID from the list of UUIDs provided with the instructions, in the order given, and never invent one.  Two dicts sharing a UUID is a profile that installs unpredictably, so use each one exactly once.',
          'PayloadIdentifier is reverse-DNS.  Each payload dict\'s identifier is the root identifier plus a distinguishing suffix, and no two identifiers in the profile are the same.',
          'PayloadDisplayName on the root is what an end user sees in System Settings, and some MDMs use it as the profile name.  Make it human-readable and specific to what the profile does.',
          // Structure.
          'Put every key for one payload domain in a single dict inside PayloadContent.  Do not emit several dicts with the same PayloadType.',


        ],
      },

      'ddm': {
        description: 'Apple DDM declaration in JSON format that enforces OS settings on macOS devices',
        firstPartySettingDescription: 'a key in an Apple-published declaration type',
        providedSchema: DDM_DECLARATION_SCHEMA_V1,
        providedSchemaDescription: `Provided context: every declaration Apple publishes, and every key each one accepts.  Format is \`Key:type\`,
where \`*\` marks a required key, \`(a|b|c)\` lists the allowed values, \`(min-max)\` gives the allowed range,
\`{...}\` is a nested dictionary, and \`[]\` is an array.

This is the complete set.  A declaration type or key that does not appear here does not exist.`,
        references: [
          'Apple DDM declaration types, keys, and values: provided in full below -- there is no other set.',
        ],
        rules: [
          'The declaration is JSON, not XML.  Include Type, Identifier, and Payload.',
          'Type must be a real declaration type, such as com.apple.configuration.passcode.settings.',
          // Key naming.  The most common DDM defect: JSON habit plus .mobileconfig bleed-through.
          'Every key inside Payload is PascalCase, with the first letter capitalized: RequirePasscode, MinimumLength, RequireAlphanumericPasscode.  A key with a lowercase first letter is an unknown key -- it is accepted as a typo and enforces nothing.',
          'Declarations do not reuse .mobileconfig payload key names, and the DDM name is not the .mobileconfig name recapitalized.  In the passcode payload, forcePIN becomes RequirePasscode and minLength becomes MinimumLength.  Never carry a .mobileconfig key into a declaration and never transform one into a declaration key.',
          'If you cannot recall a declaration\'s exact Payload key names, do not guess a casing and do not convert a .mobileconfig key -- return the "couldNotGenerateProfile" shape and name the declaration type you were unsure about.',
          // Identifier.
          'Identifier is your own reverse-DNS identifier for this declaration instance, not a copy of Type.  Copying Type conflates Apple\'s namespace with yours and collides the moment a second declaration of the same type exists.',
          'Derive Identifier from the full declaration type rather than its last component, or passcode.settings and softwareupdate.settings collapse into one identifier and silently overwrite each other.',
          'Keep Identifier to 64 bytes or fewer.  Apple\'s DeclarationBase caps it, and a longer identifier is accepted by an MDM and then rejected by the device at delivery.',
        ],
      },

    };


    let promptConfig = promptConfigByProfileType[profileType];

    // Generated list of UUIDs this profile can use.
    // Note: This is generated here and sent to the LLM to prevent it from adding invalid/placeholder UUIDs
    let suppliedPayloadUuids = [];
    if(profileType === 'mobileconfig') {
      for (let i = 0; i < 10; i++) {
        suppliedPayloadUuids.push((await sails.helpers.strings.uuid()).toUpperCase());
      }
    }


    let numberedRules = sharedRules.concat(promptConfig.rules, deliveryNotesRules)
      .map((rule, idx)=>`${idx + 1}. ${rule}`)
      .join('\n    ');

    let providedSchema = '';
    if(promptConfig.providedSchema) {
      providedSchema = `
${promptConfig.providedSchemaDescription}
\`\`\`
${promptConfig.providedSchema}
\`\`\`
`;
    }

    let systemPrompt = `Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

You generate a ${promptConfig.description} from an IT admin's instructions.

Draw setting names, types, and allowed values from these published references:
${promptConfig.references.map((reference)=>`- ${reference}`).join('\n    ')}
${providedSchema}
When generating the profile:
${numberedRules}

${RESPONSE_SHAPE}`;

    let uuidsToUse = '';
    if(suppliedPayloadUuids.length > 0) {
      uuidsToUse = `
    Use these UUIDs for PayloadUUID, in the order listed: the first on the root dict, then one per dict inside
    PayloadContent.  Never invent a UUID and never use one twice.  Where one payload references another payload
    (PayloadCertificateUUID, VPNUUID and similar), repeat that payload's UUID from this list exactly.
    ${suppliedPayloadUuids.map((uuid)=>`- ${uuid}`).join('\n    ')}
`;
    }

    let userPrompt = `Given these instructions from an IT admin, generate a ${promptConfig.description}.
${uuidsToUse}
    Here are the instructions:
    \`\`\`
    ${naturalLanguageInstructions}
    \`\`\``;

    // promptConfig comes back because the action's triage call needs description and firstPartySettingDescription; suppliedPayloadUuids so a caller can check what the model was given.
    return { systemPrompt, userPrompt, promptConfig, suppliedPayloadUuids };

  }


};


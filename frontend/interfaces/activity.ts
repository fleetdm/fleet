import { ILabelSoftwareTitle } from "./label";
import { Platform } from "./platform";
import { IPolicy } from "./policy";
import { IQuery } from "./query";
import { ISchedulableQueryStats } from "./schedulable_query";
import { ITeamSummary } from "./team";
import { UserRole } from "./user";

export enum ActivityType {
  CreatedPack = "created_pack",
  DeletedPack = "deleted_pack",
  EditedPack = "edited_pack",
  CreatedPolicy = "created_policy",
  DeletedPolicy = "deleted_policy",
  EditedPolicy = "edited_policy",
  CreatedSavedQuery = "created_saved_query",
  DeletedSavedQuery = "deleted_saved_query",
  DeletedMultipleSavedQuery = "deleted_multiple_saved_query",
  EditedSavedQuery = "edited_saved_query",
  CreatedTeam = "created_team",
  DeletedTeam = "deleted_team",
  LiveQuery = "live_query",
  AppliedSpecPack = "applied_spec_pack", // fleetctl
  AppliedSpecPolicy = "applied_spec_policy", // fleetctl
  AppliedSpecSavedQuery = "applied_spec_saved_query", // fleetctl
  AppliedSpecSoftware = "applied_spec_software", // fleetctl
  AppliedSpecTeam = "applied_spec_team", // fleetctl
  EditedAgentOptions = "edited_agent_options",
  UserAddedBySSO = "user_added_by_sso",
  UserLoggedIn = "user_logged_in",
  UserFailedLogin = "user_failed_login",
  UserCreated = "created_user",
  UserDeleted = "deleted_user",
  UserChangedGlobalRole = "changed_user_global_role",
  UserDeletedGlobalRole = "deleted_user_global_role",
  UserChangedTeamRole = "changed_user_team_role",
  UserDeletedTeamRole = "deleted_user_team_role",
  FleetEnrolled = "fleet_enrolled",
  MdmEnrolled = "mdm_enrolled",
  MdmUnenrolled = "mdm_unenrolled",
  EditedMacosMinVersion = "edited_macos_min_version",
  EditedIosMinVersion = "edited_ios_min_version",
  EditedIpadosMinVersion = "edited_ipados_min_version",
  ReadHostDiskEncryptionKey = "read_host_disk_encryption_key",
  /** Note: BE not renamed (yet) from macOS even though activity is also used for iOS and iPadOS */
  CreatedAppleOSProfile = "created_macos_profile",
  /** Note: BE not renamed (yet) from macOS even though activity is also used for iOS and iPadOS */
  DeletedAppleOSProfile = "deleted_macos_profile",
  /** Note: BE not renamed (yet) from macOS even though activity is also used for iOS and iPadOS */
  EditedAppleOSProfile = "edited_macos_profile",
  AddedNdesScepProxy = "added_ndes_scep_proxy",
  DeletedNdesScepProxy = "deleted_ndes_scep_proxy",
  EditedNdesScepProxy = "edited_ndes_scep_proxy",
  AddedDigicert = "added_digicert",
  DeletedDigicert = "deleted_digicert",
  EditedDigicert = "edited_digicert",
  AddedCustomScepProxy = "added_custom_scep_proxy",
  DeletedCustomScepProxy = "deleted_custom_scep_proxy",
  EditedCustomScepProxy = "edited_custom_scep_proxy",
  CreatedWindowsProfile = "created_windows_profile",
  DeletedWindowsProfile = "deleted_windows_profile",
  EditedWindowsProfile = "edited_windows_profile",
  // Note: Both "enabled_disk_encryption" and "enabled_macos_disk_encryption" display the same
  // message. The latter is deprecated in the API but it is retained here for backwards compatibility.
  EnabledDiskEncryption = "enabled_disk_encryption",
  EnabledMacDiskEncryption = "enabled_macos_disk_encryption",
  // Note: Both "disabled_disk_encryption" and "disabled_macos_disk_encryption" display the same
  // message. The latter is deprecated in the API but it is retained here for backwards compatibility.
  DisabledDiskEncryption = "disabled_disk_encryption",
  DisabledMacDiskEncryption = "disabled_macos_disk_encryption",
  AddedBootstrapPackage = "added_bootstrap_package",
  DeletedBootstrapPackage = "deleted_bootstrap_package",
  ChangedMacOSSetupAssistant = "changed_macos_setup_assistant",
  DeletedMacOSSetupAssistant = "deleted_macos_setup_assistant",
  EnabledMacOSSetupEndUserAuth = "enabled_macos_setup_end_user_auth",
  DisabledMacOSSetupEndUserAuth = "disabled_macos_setup_end_user_auth",
  TransferredHosts = "transferred_hosts",
  EnabledWindowsMdm = "enabled_windows_mdm",
  DisabledWindowsMdm = "disabled_windows_mdm",
  EnabledGitOpsMode = "enabled_gitops_mode",
  DisabledGitOpsMode = "disabled_gitops_mode",
  EnabledWindowsMdmMigration = "enabled_windows_mdm_migration",
  DisabledWindowsMdmMigration = "disabled_windows_mdm_migration",
  RanScript = "ran_script",
  RanScriptBatch = "ran_script_batch",
  AddedScript = "added_script",
  UpdatedScript = "updated_script",
  DeletedScript = "deleted_script",
  EditedScript = "edited_script",
  EditedWindowsUpdates = "edited_windows_updates",
  LockedHost = "locked_host",
  UnlockedHost = "unlocked_host",
  WipedHost = "wiped_host",
  CreatedDeclarationProfile = "created_declaration_profile",
  DeletedDeclarationProfile = "deleted_declaration_profile",
  EditedDeclarationProfile = "edited_declaration_profile",
  ResentConfigurationProfile = "resent_configuration_profile",
  ResentConfigurationProfileBatch = "resent_configuration_profile_batch",
  AddedSoftware = "added_software",
  EditedSoftware = "edited_software",
  DeletedSoftware = "deleted_software",
  InstalledSoftware = "installed_software",
  UninstalledSoftware = "uninstalled_software",
  EnabledVpp = "enabled_vpp",
  DisabledVpp = "disabled_vpp",
  AddedAppStoreApp = "added_app_store_app",
  EditedAppStoreApp = "edited_app_store_app",
  DeletedAppStoreApp = "deleted_app_store_app",
  InstalledAppStoreApp = "installed_app_store_app",
  EnabledActivityAutomations = "enabled_activity_automations",
  EditedActivityAutomations = "edited_activity_automations",
  DisabledActivityAutomations = "disabled_activity_automations",
  CanceledRunScript = "canceled_run_script",
  CanceledInstallAppStoreApp = "canceled_install_app_store_app",
  CanceledInstallSoftware = "canceled_install_software",
  CanceledUninstallSoftware = "canceled_uninstall_software",
  EnabledAndroidMdm = "enabled_android_mdm",
  DisabledAndroidMdm = "disabled_android_mdm",
  ConfiguredMSEntraConditionalAccess = "added_conditional_access_microsoft",
  DeletedMSEntraConditionalAccess = "deleted_conditional_access_microsoft",
  // enable/disable above feature for a team
  EnabledConditionalAccessAutomations = "enabled_conditional_access_automations",
  DisabledConditionalAccessAutomations = "disabled_conditional_access_automations",
}

/** This is a subset of ActivityType that are shown only for the host past activities */
export type IHostPastActivityType =
  | ActivityType.RanScript
  | ActivityType.LockedHost
  | ActivityType.WipedHost
  | ActivityType.ReadHostDiskEncryptionKey
  | ActivityType.UnlockedHost
  | ActivityType.InstalledSoftware
  | ActivityType.UninstalledSoftware
  | ActivityType.InstalledAppStoreApp
  | ActivityType.CanceledRunScript
  | ActivityType.CanceledInstallAppStoreApp
  | ActivityType.CanceledInstallSoftware
  | ActivityType.CanceledUninstallSoftware;

/** This is a subset of ActivityType that are shown only for the host upcoming activities */
export type IHostUpcomingActivityType =
  | ActivityType.RanScript
  | ActivityType.InstalledSoftware
  | ActivityType.UninstalledSoftware
  | ActivityType.InstalledAppStoreApp;

export interface IActivity {
  created_at: string;
  id: number;
  actor_full_name: string;
  actor_id: number;
  actor_gravatar: string;
  actor_email?: string;
  type: ActivityType;
  fleet_initiated: boolean;
  details?: IActivityDetails;
}

export type IHostPastActivity = Omit<IActivity, "type" | "details"> & {
  type: IHostPastActivityType;
  details: IActivityDetails;
};

export type IHostUpcomingActivity = Omit<
  IActivity,
  "id" | "type" | "details"
> & {
  uuid: string;
  type: IHostUpcomingActivityType;
  details: IActivityDetails;
};

export interface IActivityDetails {
  /** Useful for passing this data into an activity details modal */
  created_at?: string;
  app_store_id?: number;
  bootstrap_package_name?: string;
  batch_execution_id?: string;
  command_uuid?: string;
  deadline_days?: number;
  deadline?: string;
  email?: string;
  global?: boolean;
  grace_period_days?: number;
  host_display_name?: string;
  host_display_names?: string[];
  host_id?: number;
  host_ids?: number[];
  host_count?: number;
  host_platform?: string;
  host_serial?: string;
  install_uuid?: string;
  installed_from_dep?: boolean;
  labels_exclude_any?: ILabelSoftwareTitle[];
  labels_include_any?: ILabelSoftwareTitle[];
  location?: string; // name of location associated with VPP token
  mdm_platform?: "microsoft" | "apple";
  minimum_version?: string;
  name?: string;
  pack_id?: number;
  pack_name?: string;
  platform?: Platform; // software platform
  policy_id?: number;
  policy_name?: string;
  profile_identifier?: string;
  profile_name?: string;
  public_ip?: string;
  query_id?: number;
  query_ids?: number[];
  query_name?: string;
  query_sql?: string;
  role?: UserRole;
  script_execution_id?: string;
  script_name?: string;
  self_service?: boolean;
  software_package?: string;
  software_title_id?: number;
  software_title?: string;
  specs?: IQuery[] | IPolicy[];
  stats?: ISchedulableQueryStats;
  status?: string;
  targets_count?: number;
  team_id?: number | null;
  team_name?: string | null;
  teams?: ITeamSummary[];
  user_email?: string;
  user_id?: number;
  webhook_url?: string;
}

export const ACTIVITY_DISPLAY_NAME_MAP: Record<ActivityType, string> = {
  added_app_store_app: "Added App Store (VPP) app",
  added_bootstrap_package: "Added bootstrap package",
  added_conditional_access_microsoft: "Added conditional access - Microsoft",
  added_custom_scep_proxy: "Added certificate authority (CA) - custom SCEP",
  added_digicert: "Added certificate authority (CA) - DigiCert",
  added_ndes_scep_proxy: "Added certificate authority (CA) - NDES",
  added_script: "Added script",
  added_software: "Added software",
  applied_spec_pack: "GitOps - edited packs",
  applied_spec_policy: "GitOps - edited policies",
  applied_spec_saved_query: "GitOps - edited queries",
  applied_spec_team: "GitOps - edited teams",
  applied_spec_software: "GitOps - edited software",
  canceled_install_app_store_app:
    "Canceled activity - install App Store (VPP) app",
  canceled_install_software: "Canceled activity - install software",
  canceled_run_script: "Canceled activity - run script",
  canceled_uninstall_software: "Canceled activity - uninstall software",
  changed_macos_setup_assistant: "Edited macOS automatic enrollment profile",
  changed_user_global_role: "Edited user's role - global",
  changed_user_team_role: "Edited user's role - team",
  created_declaration_profile: "Added declaration (DDM) profile",
  created_macos_profile:
    "Added configuration profile - Apple (macOS, iOS, iPadOS)",
  created_pack: "Created pack",
  created_policy: "Created policy",
  created_saved_query: "Added query",
  created_team: "Added team",
  created_user: "Added user",
  created_windows_profile: "Added configuration profile - Windows",
  deleted_app_store_app: "Deleted App Store (VPP) app",
  deleted_bootstrap_package: "Deleted bootstrap package",
  deleted_conditional_access_microsoft:
    "Deleted conditional access - Microsoft",
  deleted_custom_scep_proxy: "Deleted certificate authority (CA) - custom SCEP",
  deleted_declaration_profile: "Deleted declaration (DDM) profile",
  deleted_digicert: "Deleted certificate authority (CA) - DigiCert",
  deleted_macos_profile:
    "Deleted configuration profile - Apple (macOS, iOS, iPadOS)",
  deleted_macos_setup_assistant: "Deleted macOS automatic enrollment profile",
  deleted_multiple_saved_query: "Bulk deleted queries",
  deleted_ndes_scep_proxy: "Deleted certificate authority (CA) - NDES",
  deleted_pack: "Deleted pack",
  deleted_policy: "Deleted policy",
  deleted_saved_query: "Deleted query",
  deleted_script: "Deleted script",
  deleted_software: "Deleted software",
  deleted_team: "Deleted team",
  deleted_user: "Deleted user",
  deleted_user_global_role: "Deleted user's role - global",
  deleted_user_team_role: "Deleted user's role - team",
  deleted_windows_profile: "Deleted configuration profile - Windows",
  disabled_activity_automations: "Disabled activity automations",
  disabled_android_mdm: "Turned off Android MDM",
  disabled_conditional_access_automations:
    "Disabled conditional access automations",
  disabled_gitops_mode: "Disabled GitOps mode",
  disabled_disk_encryption: "Turned off disk encryption",
  disabled_macos_disk_encryption: "Turned off disk encryption",
  disabled_macos_setup_end_user_auth:
    "Turned off end user authentication (setup experience)",
  disabled_vpp: "Disabled Volume Purchasing Program (VPP)",
  disabled_windows_mdm: "Turned off Windows MDM",
  disabled_windows_mdm_migration: "Turned off Windows MDM migration",
  edited_activity_automations: "Edited activity automations",
  edited_agent_options: "Edited agent options",
  edited_app_store_app: "Edited App Store (VPP) app",
  edited_custom_scep_proxy: "Edited certificate authority (CA) - custom SCEP",
  edited_declaration_profile: "GitOps - edites declaration (DDM) profiles",
  edited_digicert: "Edited certificate authority (CA) - DigiCert",
  edited_ios_min_version: "OS updates - edited iOS",
  edited_ipados_min_version: "OS updates - edited iPadOS",
  edited_macos_min_version: "OS updates - edited macOS",
  edited_macos_profile:
    "GitOps - edited configuration profiles - Apple (macOS, iOS, iPadOS)",
  edited_ndes_scep_proxy: "Edited certificate authority (CA) - NDES",
  edited_pack: "Edited pack",
  edited_policy: "Edited policy",
  edited_saved_query: "Edited query",
  edited_script: "Edited script",
  edited_software: "Edited software",
  edited_windows_profile: "GitOps - edited configuration profiles - Windows",
  edited_windows_updates: "OS updates - edited Windows",
  enabled_activity_automations: "Enabled activity automations",
  enabled_android_mdm: "Turned on Android MDM",
  enabled_conditional_access_automations:
    "Enabled conditional access automations",
  enabled_gitops_mode: "Enabled GitOps mode",
  enabled_disk_encryption: "Turned on disk encryption",
  enabled_macos_disk_encryption: "Turned on disk encryption",
  enabled_macos_setup_end_user_auth:
    "Turned on end user authentication (setup experience)",
  enabled_vpp: "Enabled Volume Purchasing Program (VPP)",
  enabled_windows_mdm: "Turned on Windows MDM",
  enabled_windows_mdm_migration: "Turned on Windows MDM migration",
  fleet_enrolled: "Host enrolled",
  installed_app_store_app: "Installed App Store (VPP) app",
  installed_software: "Install software",
  live_query: "Ran live query",
  locked_host: "Locked host",
  mdm_enrolled: "MDM turned on",
  mdm_unenrolled: "MDM turned off",
  ran_script: "Ran script",
  ran_script_batch: "Bulk ran script",
  read_host_disk_encryption_key: "Viewed disk encryption key",
  resent_configuration_profile: "Resent configuration profile",
  resent_configuration_profile_batch: "Bulk resent configuration profile",
  transferred_hosts: "Transferred hosts",
  uninstalled_software: "Uninstall software",
  unlocked_host: "Unlocked host",
  updated_script: "Edited script",
  user_added_by_sso: "Added user via JIT",
  user_failed_login: "User login - failed",
  user_logged_in: "User login - success",
  wiped_host: "Wiped host",
};

import React from "react";
import { cloneDeep } from "lodash";

import { IDropdownOption } from "interfaces/dropdownOption";
import {
  isLinuxLike,
  isAppleDevice,
  isMacOS,
  isMobilePlatform,
  isAndroid,
  isIPadOrIPhone,
} from "interfaces/platform";
import { isScriptSupportedPlatform } from "interfaces/script";
import {
  isAutomaticDeviceEnrollment,
  isBYODAccountDrivenUserEnrollment,
  MdmEnrollmentStatus,
} from "interfaces/mdm";

import {
  HostMdmDeviceStatusUIState,
  isDeviceStatusUpdating,
} from "../../helpers";

const DEFAULT_OPTIONS = [
  {
    label: "Transfer",
    value: "transfer",
    disabled: false,
    premiumOnly: true,
  },
  {
    label: "Live report",
    value: "query",
    disabled: false,
  },
  {
    label: "Run script",
    value: "runScript",
    disabled: false,
  },
  {
    label: "Show disk encryption key",
    value: "diskEncryption",
    disabled: false,
  },
  {
    label: "Show Recovery Lock password",
    value: "recoveryLockPassword",
    disabled: false,
  },
  {
    label: "Show managed account",
    value: "managedAccount",
    disabled: false,
  },
  {
    label: "Turn off MDM",
    value: "mdmOff",
    disabled: false,
  },
  {
    label: "Lock",
    value: "lock",
    disabled: false,
  },
  {
    label: "Wipe",
    value: "wipe",
    disabled: false,
  },
  {
    label: "Unlock",
    value: "unlock",
    disabled: false,
  },
  {
    label: "Clear passcode",
    value: "clearPasscode",
    disabled: false,
  },
  {
    label: "Delete",
    disabled: false,
    value: "delete",
  },
] as const;

interface IHostActionConfigOptions {
  hostPlatform: string;
  hostCpuType: string;
  isPremiumTier: boolean;
  isGlobalAdmin: boolean;
  isGlobalMaintainer: boolean;
  isGlobalObserver: boolean;
  isGlobalTechnician: boolean;
  isTeamAdmin: boolean;
  isTeamMaintainer: boolean;
  isTeamTechnician: boolean;
  isTeamObserver: boolean;
  isHostOnline: boolean;
  isEnrolledInMdm: boolean;
  isConnectedToFleetMdm?: boolean;
  isMacMdmEnabledAndConfigured: boolean;
  isWindowsMdmEnabledAndConfigured: boolean;
  isAndroidMdmEnabledAndConfigured: boolean;
  doesStoreEncryptionKey: boolean;
  hostMdmDeviceStatus: HostMdmDeviceStatusUIState;
  hostScriptsEnabled: boolean | null;
  scriptsGloballyDisabled: boolean | undefined;
  isPrimoMode: boolean;
  hostMdmEnrollmentStatus: MdmEnrollmentStatus | null;
  isRecoveryLockPasswordEnabled: boolean;
  diskEncryptionProfileStatus: string | undefined;
  recoveryLockPasswordAvailable: boolean;
  isManagedLocalAccountEnabled: boolean;
  managedAccountStatus: string | null | undefined;
  managedAccountPasswordAvailable: boolean;
}

const canTransferTeam = (config: IHostActionConfigOptions) => {
  const {
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isPrimoMode,
  } = config;
  return isPremiumTier && (isGlobalAdmin || isGlobalMaintainer) && !isPrimoMode;
};

const canTurnOffMdm = (config: IHostActionConfigOptions) => {
  const {
    hostPlatform,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    isEnrolledInMdm,
    isConnectedToFleetMdm,
    isMacMdmEnabledAndConfigured,
    isAndroidMdmEnabledAndConfigured,
  } = config;
  return (
    ((isAndroid(hostPlatform) && isAndroidMdmEnabledAndConfigured) ||
      (isAppleDevice(hostPlatform) && isMacMdmEnabledAndConfigured)) &&
    isEnrolledInMdm &&
    isConnectedToFleetMdm &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer)
  );
};

const canQueryHost = ({ hostPlatform }: IHostActionConfigOptions) => {
  // cannot query iOS, iPadOS, or Android hosts
  return !isMobilePlatform(hostPlatform);
};

const canLockHost = ({
  isPremiumTier,
  hostPlatform,
  isMacMdmEnabledAndConfigured,
  isEnrolledInMdm,
  isConnectedToFleetMdm,
  isGlobalAdmin,
  isGlobalMaintainer,
  isTeamAdmin,
  isTeamMaintainer,
  hostMdmDeviceStatus,
  hostMdmEnrollmentStatus,
}: IHostActionConfigOptions) => {
  // apple device hosts can be locked if they are enrolled in MDM and the MDM is enabled
  const isLockableMacOSDevice =
    hostPlatform === "darwin" &&
    isConnectedToFleetMdm &&
    isMacMdmEnabledAndConfigured &&
    isEnrolledInMdm;

  // ios and ipad devices can be locked if they are company owned enrollment in MDM,
  // meaning they have to be enrolled via automated device enrollment (ADE)
  const isLockableIosOrIpadDevice =
    isIPadOrIPhone(hostPlatform) &&
    isAutomaticDeviceEnrollment(hostMdmEnrollmentStatus) &&
    isConnectedToFleetMdm &&
    isMacMdmEnabledAndConfigured &&
    isEnrolledInMdm;

  return (
    isPremiumTier &&
    !isAndroid(hostPlatform) &&
    hostMdmDeviceStatus === "unlocked" &&
    (hostPlatform === "windows" ||
      isLinuxLike(hostPlatform) ||
      isLockableMacOSDevice ||
      isLockableIosOrIpadDevice) &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer)
  );
};

const canWipeHost = ({
  isPremiumTier,
  isGlobalAdmin,
  isGlobalMaintainer,
  isTeamAdmin,
  isTeamMaintainer,
  isConnectedToFleetMdm,
  isEnrolledInMdm,
  isMacMdmEnabledAndConfigured,
  isWindowsMdmEnabledAndConfigured,
  hostPlatform,
  hostMdmDeviceStatus,
  hostMdmEnrollmentStatus,
}: IHostActionConfigOptions) => {
  const hostMdmEnabled =
    (isAppleDevice(hostPlatform) && isMacMdmEnabledAndConfigured) ||
    (hostPlatform === "windows" && isWindowsMdmEnabledAndConfigured);

  // Windows and Apple devices (i.e. macOS, iOS, iPadOS, windows) have the same
  // conditions and can be wiped if they are enrolled in MDM and the MDM is enabled.
  const canWipeWindowsOrAppleOS =
    hostMdmEnabled && isConnectedToFleetMdm && isEnrolledInMdm;

  // there is a special case for iOS and iPadOS devices that are account driven enrolled
  // in MDM. These hosts cannot be wiped.
  const isAccountDrivenEnrolledIosOrIpadosDevice =
    isIPadOrIPhone(hostPlatform) &&
    isBYODAccountDrivenUserEnrollment(hostMdmEnrollmentStatus);

  return (
    isPremiumTier &&
    !isAndroid(hostPlatform) &&
    !isAccountDrivenEnrolledIosOrIpadosDevice &&
    hostMdmDeviceStatus === "unlocked" &&
    (isLinuxLike(hostPlatform) || canWipeWindowsOrAppleOS) &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer)
  );
};

const canUnlock = ({
  isPremiumTier,
  isGlobalAdmin,
  isGlobalMaintainer,
  isTeamAdmin,
  isTeamMaintainer,
  isConnectedToFleetMdm,
  isEnrolledInMdm,
  isMacMdmEnabledAndConfigured,
  hostPlatform,
  hostMdmDeviceStatus,
}: IHostActionConfigOptions) => {
  // apple device hosts can be unlocked if they are enrolled in the Fleet MDM and the
  // MDM is enabled and configured.
  const canUnlockApple =
    isAppleDevice(hostPlatform) &&
    isConnectedToFleetMdm &&
    isMacMdmEnabledAndConfigured &&
    isEnrolledInMdm;

  // "unlocking" for a macos devices host means that somebody saw the unlock pin, but we
  // shouldn't prevent users from trying to see the pin again so we still want to show
  // the unlock option when macos hosts are unlocking. This is not the same for
  // ios/ipad devices.
  const isValidState =
    (hostMdmDeviceStatus === "unlocking" && isMacOS(hostPlatform)) ||
    hostMdmDeviceStatus === "locked";

  return (
    isPremiumTier &&
    !isAndroid(hostPlatform) &&
    isValidState &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer) &&
    (canUnlockApple || hostPlatform === "windows" || isLinuxLike(hostPlatform))
  );
};

const canDeleteHost = (config: IHostActionConfigOptions) => {
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = config;
  return isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;
};

const canShowDiskEncryption = (config: IHostActionConfigOptions) => {
  const {
    isPremiumTier,
    isConnectedToFleetMdm,
    doesStoreEncryptionKey,
    hostPlatform,
  } = config;
  if (!isPremiumTier) {
    return false;
  }
  if (isMobilePlatform(hostPlatform)) {
    return false;
  }
  // For Apple devices, the encryption key is only available when connected to Fleet MDM
  if (isAppleDevice(hostPlatform) && !isConnectedToFleetMdm) {
    return false;
  }
  return doesStoreEncryptionKey;
};

const canShowRecoveryLockPassword = (config: IHostActionConfigOptions) => {
  const {
    isPremiumTier,
    isConnectedToFleetMdm,
    hostPlatform,
    hostCpuType,
    isRecoveryLockPasswordEnabled,
    recoveryLockPasswordAvailable,
  } = config;
  if (!isPremiumTier) {
    return false;
  }
  if (hostPlatform !== "darwin") {
    return false;
  }
  // permissive by default - only remove option if we know the host's cpu type and it is not arm64
  if (hostCpuType !== "" && !hostCpuType.includes("arm64")) {
    return false;
  }
  if (!isConnectedToFleetMdm) {
    return false;
  }
  // A password may exist on a host whose current team has the setting
  // disabled (e.g., the host was locked under a different team's policy).
  // Don't hide an existing password just because the team setting flipped.
  return isRecoveryLockPasswordEnabled || recoveryLockPasswordAvailable;
};

const canShowManagedAccount = (config: IHostActionConfigOptions) => {
  const {
    isPremiumTier,
    isConnectedToFleetMdm,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    hostPlatform,
    hostMdmEnrollmentStatus,
    isManagedLocalAccountEnabled,
  } = config;
  if (!isPremiumTier) return false;
  if (hostPlatform !== "darwin") return false;
  if (!isConnectedToFleetMdm) return false;
  if (!isAutomaticDeviceEnrollment(hostMdmEnrollmentStatus)) return false;
  if (!isManagedLocalAccountEnabled && !config.managedAccountStatus) {
    return false;
  }
  return isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;
};

const canClearPasscode = (config: IHostActionConfigOptions) => {
  if (!config.isPremiumTier) {
    return false;
  }

  if (!isIPadOrIPhone(config.hostPlatform)) {
    return false;
  }

  if (!config.isEnrolledInMdm) {
    return false;
  }

  if (!config.isConnectedToFleetMdm) {
    return false;
  }

  if (!config.isMacMdmEnabledAndConfigured) {
    return false;
  }

  if (
    config.hostMdmEnrollmentStatus !== "On (company-owned)" &&
    config.hostMdmEnrollmentStatus !== "On (automatic)" &&
    config.hostMdmEnrollmentStatus !== "On (manual)"
  ) {
    return false;
  }

  return (
    config.isGlobalAdmin ||
    config.isGlobalMaintainer ||
    config.isTeamAdmin ||
    config.isTeamMaintainer
  );
};

const canRunScript = ({
  hostPlatform,
  isGlobalAdmin,
  isGlobalMaintainer,
  isGlobalTechnician,
  isTeamAdmin,
  isTeamMaintainer,
  isTeamTechnician,
}: IHostActionConfigOptions) => {
  // Scripts globally disabled, shown as disabled by modifyOptions

  return (
    (isGlobalAdmin ||
      isGlobalMaintainer ||
      isGlobalTechnician ||
      isTeamAdmin ||
      isTeamMaintainer ||
      isTeamTechnician) &&
    isScriptSupportedPlatform(hostPlatform)
  );
};

const removeUnavailableOptions = (
  options: IDropdownOption[],
  config: IHostActionConfigOptions
) => {
  if (!canTransferTeam(config)) {
    options = options.filter((option) => option.value !== "transfer");
  }

  if (!canQueryHost(config)) {
    options = options.filter((option) => option.value !== "query");
  }

  if (!canShowDiskEncryption(config)) {
    options = options.filter((option) => option.value !== "diskEncryption");
  }

  if (!canShowRecoveryLockPassword(config)) {
    options = options.filter(
      (option) => option.value !== "recoveryLockPassword"
    );
  }

  if (!canShowManagedAccount(config)) {
    options = options.filter((option) => option.value !== "managedAccount");
  }

  if (!canClearPasscode(config)) {
    options = options.filter((option) => option.value !== "clearPasscode");
  }

  if (!canTurnOffMdm(config)) {
    options = options.filter((option) => option.value !== "mdmOff");
  }

  if (!canDeleteHost(config)) {
    options = options.filter((option) => option.value !== "delete");
  }

  if (!canRunScript(config)) {
    options = options.filter((option) => option.value !== "runScript");
  }

  if (!canLockHost(config)) {
    options = options.filter((option) => option.value !== "lock");
  }

  if (!canWipeHost(config)) {
    options = options.filter((option) => option.value !== "wipe");
  }

  if (!canUnlock(config)) {
    options = options.filter((option) => option.value !== "unlock");
  }

  // TODO: refactor to filter in one pass using predefined filters specified for each of the
  // DEFAULT_OPTIONS. Note that as currently, structured the default is to include all options.
  // This is a bit confusing since we remove options instead of add options

  return options;
};

// Available tooltips for disabled options
export const getDropdownOptionTooltipContent = (
  value: string | number,
  isHostOnline?: boolean,
  scriptsGloballyDisabled?: boolean
) => {
  if (value === "runScript" && scriptsGloballyDisabled) {
    return <>Running scripts is disabled in organization settings.</>;
  }

  const tooltipAction: Record<string, string> = {
    runScript: "run scripts on",
    wipe: "wipe",
    lock: "lock",
    unlock: "unlock",
  };
  if (tooltipAction[value]) {
    return (
      <>
        To {tooltipAction[value]} this host, deploy the
        <br />
        fleetd agent with --enable-scripts and
        <br />
        refetch host vitals
      </>
    );
  }
  if (!isHostOnline && value === "query") {
    return <>You can&apos;t run a live report on an offline host.</>;
  }
  return undefined;
};

/** for ios, ipad, and android we want to display different text for mdmOff.
 * The functionality is the same, but the action is called unenroll on those platforms.
 */
const formatTurnOffOptionLabel = (
  options: IDropdownOption[],
  hostPlatform: string
) => {
  const option = options.find((opt) => opt.value === "mdmOff");
  if (option && (isIPadOrIPhone(hostPlatform) || isAndroid(hostPlatform))) {
    option.label = "Unenroll";
  }
};

const modifyOptions = (
  options: IDropdownOption[],
  {
    isHostOnline,
    hostMdmDeviceStatus,
    hostScriptsEnabled,
    hostPlatform,
    scriptsGloballyDisabled,
    diskEncryptionProfileStatus,
    recoveryLockPasswordAvailable,
    managedAccountStatus,
    managedAccountPasswordAvailable,
  }: IHostActionConfigOptions
) => {
  const disableOptions = (optionsToDisable: IDropdownOption[]) => {
    optionsToDisable.forEach((option) => {
      option.disabled = true;
      option.tooltipContent = getDropdownOptionTooltipContent(
        option.value,
        isHostOnline,
        scriptsGloballyDisabled
      );
    });
  };

  let optionsToDisable: IDropdownOption[] = [];
  // When the host is offline, always disable Query, but allow Unenroll for iOS/iPadOS and Android.
  if (!isHostOnline) {
    optionsToDisable = optionsToDisable.concat(
      options.filter((option) => option.value === "query")
    );

    // Disable "Turn off MDM" (Unenroll) when offline for all platforms except iOS/iPadOS and Android
    if (!isIPadOrIPhone(hostPlatform) && !isAndroid(hostPlatform)) {
      optionsToDisable = optionsToDisable.concat(
        options.filter((option) => option.value === "mdmOff")
      );
    }
  }

  // While device status is updating, or device is locked/wiped, disable Query and Turn off MDM
  if (
    isDeviceStatusUpdating(hostMdmDeviceStatus) ||
    hostMdmDeviceStatus === "locked" ||
    hostMdmDeviceStatus === "wiped"
  ) {
    optionsToDisable = optionsToDisable.concat(
      options.filter(
        (option) => option.value === "query" || option.value === "mdmOff"
      )
    );
  }

  // Disable run script feature if scripts are globally disabled
  if (scriptsGloballyDisabled) {
    optionsToDisable = optionsToDisable.concat(
      options.filter((option) => option.value === "runScript")
    );
  }

  // null intentionally excluded from this condition:
  // scripts_enabled === null means this agent is not an orbit agent, or this agent is version
  // <=1.23.0 which is not collecting the scripts enabled info
  // in each of these cases, we maintain these options
  if (hostScriptsEnabled === false) {
    optionsToDisable = optionsToDisable.concat(
      options.filter((option) => option.value === "runScript")
    );
    if (isLinuxLike(hostPlatform)) {
      optionsToDisable = optionsToDisable.concat(
        options.filter(
          (option) =>
            option.value === "lock" ||
            option.value === "unlock" ||
            option.value === "wipe"
        )
      );
    }
    if (hostPlatform === "windows") {
      optionsToDisable = optionsToDisable.concat(
        options.filter(
          (option) => option.value === "lock" || option.value === "unlock"
        )
      );
    }
  }
  if (
    diskEncryptionProfileStatus === "pending" ||
    diskEncryptionProfileStatus === "failed"
  ) {
    const diskEncOption = options.find(
      (option) => option.value === "diskEncryption"
    );
    if (diskEncOption) {
      diskEncOption.disabled = true;
      diskEncOption.tooltipContent = (
        <>
          Disk encryption key is unavailable
          <br />
          while pending or has failed.
        </>
      );
    }
  }

  if (!recoveryLockPasswordAvailable) {
    const rlpOption = options.find(
      (option) => option.value === "recoveryLockPassword"
    );
    if (rlpOption) {
      rlpOption.disabled = true;
      rlpOption.tooltipContent = (
        <>
          Recovery Lock password is unavailable
          <br />
          while pending or has failed.
        </>
      );
    }
  }

  // Gate on password_available rather than status === "verified" — a row whose
  // status is "pending" because of a recent view (or a deferred rotation
  // waiting on UUID capture) still has a viewable password. Mirrors the
  // backend gate in GetHostManagedAccountPassword.
  if (!managedAccountPasswordAvailable) {
    const managedAccountOption = options.find(
      (option) => option.value === "managedAccount"
    );
    if (managedAccountOption) {
      managedAccountOption.disabled = true;
      if (managedAccountStatus === "pending") {
        // No password yet — the AccountConfiguration command hasn't been acked.
        managedAccountOption.tooltipContent = (
          <>
            The managed account is still being
            <br />
            created.
          </>
        );
      } else if (managedAccountStatus === "failed") {
        managedAccountOption.tooltipContent = (
          <>
            The managed account failed to be
            <br />
            created. It will retry at the next enrollment.
          </>
        );
      } else {
        // status is null/undefined — no record exists for this host
        managedAccountOption.tooltipContent = (
          <>
            This host will receive a managed account
            <br />
            at the next enrollment. Already enrolled
            <br />
            hosts don&apos;t get a managed account.
          </>
        );
      }
    }
  }

  const clearPasscodeOption = options.find(
    (option) => option.value === "clearPasscode"
  );
  if (
    clearPasscodeOption &&
    ["locked", "locking", "unlocking", "locating"].includes(hostMdmDeviceStatus)
  ) {
    clearPasscodeOption.disabled = true;
    clearPasscodeOption.tooltipContent =
      "Clear passcode is unavailable while host is in Lost Mode.";
  } else if (
    clearPasscodeOption &&
    ["wiped", "wiping"].includes(hostMdmDeviceStatus)
  ) {
    clearPasscodeOption.disabled = true;
    clearPasscodeOption.tooltipContent =
      "Clear passcode is unavailable while host is pending wipe.";
  }
  disableOptions(optionsToDisable);
  formatTurnOffOptionLabel(options, hostPlatform);
  return options;
};

/**
 * Generates the host actions options depending on the configuration. There are
 * many variations of the options that are shown/not shown or disabled/enabled
 * which are all controlled by the configurations options argument.
 */
export const generateHostActionOptions = (config: IHostActionConfigOptions) => {
  // deep clone to always start with a fresh copy of the default options.
  let options: IDropdownOption[] = cloneDeep([...DEFAULT_OPTIONS]);
  options = removeUnavailableOptions(options, config);

  if (options.length === 0) return options;

  options = modifyOptions(options, config);

  return options;
};

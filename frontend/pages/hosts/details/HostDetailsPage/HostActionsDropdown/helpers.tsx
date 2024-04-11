import React from "react";
import { cloneDeep } from "lodash";

import { IDropdownOption } from "interfaces/dropdownOption";
import { isLinuxLike } from "interfaces/platform";
import { isScriptSupportedPlatform } from "interfaces/script";

import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";

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
    label: "Query",
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
    label: "Delete",
    disabled: false,
    value: "delete",
  },
] as const;

// eslint-disable-next-line import/prefer-default-export
interface IHostActionConfigOptions {
  hostPlatform: string;
  isPremiumTier: boolean;
  isGlobalAdmin: boolean;
  isGlobalMaintainer: boolean;
  isGlobalObserver: boolean;
  isTeamAdmin: boolean;
  isTeamMaintainer: boolean;
  isTeamObserver: boolean;
  isHostOnline: boolean;
  isEnrolledInMdm: boolean;
  isFleetMdm: boolean;
  isMacMdmEnabledAndConfigured: boolean;
  isWindowsMdmEnabledAndConfigured: boolean;
  doesStoreEncryptionKey: boolean;
  isSandboxMode: boolean;
  hostMdmDeviceStatus: HostMdmDeviceStatusUIState;
  hostScriptsEnabled: boolean | null;
}

const canTransferTeam = (config: IHostActionConfigOptions) => {
  const { isPremiumTier, isGlobalAdmin, isGlobalMaintainer } = config;
  return isPremiumTier && (isGlobalAdmin || isGlobalMaintainer);
};

const canEditMdm = (config: IHostActionConfigOptions) => {
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    isEnrolledInMdm,
    isFleetMdm,
    isMacMdmEnabledAndConfigured,
  } = config;
  return (
    config.hostPlatform === "darwin" &&
    isMacMdmEnabledAndConfigured &&
    isEnrolledInMdm &&
    isFleetMdm &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer)
  );
};

const canLockHost = ({
  isPremiumTier,
  hostPlatform,
  isMacMdmEnabledAndConfigured,
  isEnrolledInMdm,
  isFleetMdm,
  isGlobalAdmin,
  isGlobalMaintainer,
  isTeamAdmin,
  isTeamMaintainer,
  hostMdmDeviceStatus,
}: IHostActionConfigOptions) => {
  // macOS hosts can be locked if they are enrolled in MDM and the MDM is enabled
  const canLockDarwin =
    hostPlatform === "darwin" &&
    isFleetMdm &&
    isMacMdmEnabledAndConfigured &&
    isEnrolledInMdm;

  return (
    isPremiumTier &&
    hostMdmDeviceStatus === "unlocked" &&
    (hostPlatform === "windows" ||
      isLinuxLike(hostPlatform) ||
      canLockDarwin) &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer)
  );
};

const canWipeHost = ({
  isPremiumTier,
  isGlobalAdmin,
  isGlobalMaintainer,
  isGlobalObserver,
  isTeamAdmin,
  isTeamMaintainer,
  isTeamObserver,
  isFleetMdm,
  isEnrolledInMdm,
  isMacMdmEnabledAndConfigured,
  isWindowsMdmEnabledAndConfigured,
  hostPlatform,
  hostMdmDeviceStatus,
}: IHostActionConfigOptions) => {
  const hostMdmEnabled =
    (hostPlatform === "darwin" && isMacMdmEnabledAndConfigured) ||
    (hostPlatform === "windows" && isWindowsMdmEnabledAndConfigured);

  // macOS and Windows hosts have the same conditions and can be wiped if they
  // are enrolled in MDM and the MDM is enabled.
  const canWipeMacOrWindows = hostMdmEnabled && isFleetMdm && isEnrolledInMdm;

  return (
    isPremiumTier &&
    hostMdmDeviceStatus === "unlocked" &&
    (isLinuxLike(hostPlatform) || canWipeMacOrWindows) &&
    (isGlobalAdmin ||
      isGlobalMaintainer ||
      isGlobalObserver ||
      isTeamAdmin ||
      isTeamMaintainer ||
      isTeamObserver)
  );
};

const canUnlock = ({
  isPremiumTier,
  isGlobalAdmin,
  isGlobalMaintainer,
  isTeamAdmin,
  isTeamMaintainer,
  isFleetMdm,
  isEnrolledInMdm,
  isMacMdmEnabledAndConfigured,
  hostPlatform,
  hostMdmDeviceStatus,
}: IHostActionConfigOptions) => {
  const canUnlockDarwin =
    hostPlatform === "darwin" &&
    isFleetMdm &&
    isMacMdmEnabledAndConfigured &&
    isEnrolledInMdm;

  // "unlocking" for a macOS host means that somebody saw the unlock pin, but
  // shouldn't prevent users from trying to see the pin again, which is
  // considered an "unlock"
  const isValidState =
    (hostMdmDeviceStatus === "unlocking" && hostPlatform === "darwin") ||
    hostMdmDeviceStatus === "locked";

  return (
    isPremiumTier &&
    isValidState &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer) &&
    (canUnlockDarwin || hostPlatform === "windows" || isLinuxLike(hostPlatform))
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
  const { isPremiumTier, doesStoreEncryptionKey } = config;
  return isPremiumTier && doesStoreEncryptionKey;
};

const canRunScript = ({
  hostPlatform,
  isGlobalAdmin,
  isGlobalMaintainer,
  isGlobalObserver,
  isTeamAdmin,
  isTeamMaintainer,
  isTeamObserver,
}: IHostActionConfigOptions) => {
  return (
    (isGlobalAdmin ||
      isGlobalMaintainer ||
      isGlobalObserver ||
      isTeamAdmin ||
      isTeamMaintainer ||
      isTeamObserver) &&
    isScriptSupportedPlatform(hostPlatform)
  );
};

const filterOutOptions = (
  options: IDropdownOption[],
  config: IHostActionConfigOptions
) => {
  if (!canTransferTeam(config)) {
    options = options.filter((option) => option.value !== "transfer");
  }

  if (!canShowDiskEncryption(config)) {
    options = options.filter((option) => option.value !== "diskEncryption");
  }

  if (!canEditMdm(config)) {
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
  // DEFAULT_OPTIONS. Note that as currently, structured the default is to include all options. For
  // example, "Query" is implicitly included by default because there is no equivalent `canQuery`
  // filter being applied here. This is a bit confusing since

  return options;
};

const setOptionsAsDisabled = (
  options: IDropdownOption[],
  {
    isHostOnline,
    isSandboxMode,
    hostMdmDeviceStatus,
    hostScriptsEnabled,
  }: IHostActionConfigOptions
) => {
  // Available tooltips for disabled options
  const disabledTooltipContent = (value: string | number) => {
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
          fleetd agent with --enable-scripts
        </>
      );
    }
    if (!isHostOnline && value === "query") {
      return <>You can&apos;t query an offline host.</>;
    }
  };

  const disableOptions = (optionsToDisable: IDropdownOption[]) => {
    optionsToDisable.forEach((option) => {
      option.disabled = true;
      option.disabledTooltipContent = disabledTooltipContent(option.value);
    });
  };

  let optionsToDisable: IDropdownOption[] = [];
  if (
    !isHostOnline ||
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

  if (!hostScriptsEnabled) {
    optionsToDisable = optionsToDisable.concat(
      options.filter((option) => option.value === "runScript")
    );
  }
  if (isSandboxMode) {
    optionsToDisable = optionsToDisable.concat(
      options.filter((option) => option.value === "transfer")
    );
  }

  disableOptions(optionsToDisable);
  return options;
};

/**
 * Generates the host actions options depending on the configuration. There are
 * many variations of the options that are shown/not shown or disabled/enabled
 * which are all controlled by the configurations options argument.
 */
// eslint-disable-next-line import/prefer-default-export
export const generateHostActionOptions = (config: IHostActionConfigOptions) => {
  // deep clone to always start with a fresh copy of the default options.
  let options: IDropdownOption[] = cloneDeep([...DEFAULT_OPTIONS]);
  options = filterOutOptions(options, config);

  if (options.length === 0) return options;

  options = setOptionsAsDisabled(options, config);

  if (config.isSandboxMode) {
    const premiumOnlyOptions: IDropdownOption[] = options.filter(
      (option) => !!option.premiumOnly
    );

    premiumOnlyOptions.forEach((option) => {
      option.label = (
        <span>
          {option.label}
          <PremiumFeatureIconWithTooltip
            tooltipPositionOverrides={{ leftAdj: 2 }}
          />
        </span>
      );
    });
  }

  return options;
};

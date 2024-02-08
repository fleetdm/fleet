import React from "react";
import { IDropdownOption } from "interfaces/dropdownOption";
import { cloneDeep } from "lodash";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";
import { isScriptSupportedPlatform } from "interfaces/script";

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
  isMdmEnabledAndConfigured: boolean;
  doesStoreEncryptionKey: boolean;
  isSandboxMode: boolean;
  isLocking: boolean;
  isWiping: boolean;
  isUnlocking: boolean;
  isLocked: boolean;
  isWiped: boolean;
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
    isMdmEnabledAndConfigured,
  } = config;
  return (
    config.hostPlatform === "darwin" &&
    isMdmEnabledAndConfigured &&
    isEnrolledInMdm &&
    isFleetMdm &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer)
  );
};

const canLockHost = ({
  isPremiumTier,
  hostPlatform,
  isMdmEnabledAndConfigured,
  isEnrolledInMdm,
  isFleetMdm,
  isGlobalAdmin,
  isGlobalMaintainer,
  isGlobalObserver,
  isTeamAdmin,
  isTeamMaintainer,
  isTeamObserver,
  isLocked,
}: IHostActionConfigOptions) => {
  // macOS hosts can be locked if they are enrolled in MDM and the MDM is enabled
  const canLockDarwin =
    hostPlatform === "darwin" &&
    isFleetMdm &&
    isMdmEnabledAndConfigured &&
    isEnrolledInMdm;

  return (
    isPremiumTier &&
    !isLocked &&
    (hostPlatform === "windows" || hostPlatform === "linux" || canLockDarwin) &&
    (isGlobalAdmin ||
      isGlobalMaintainer ||
      isGlobalObserver ||
      isTeamAdmin ||
      isTeamMaintainer ||
      isTeamObserver)
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
  isMdmEnabledAndConfigured,
  hostPlatform,
  isLocked,
}: IHostActionConfigOptions) => {
  // TODO: remove when we work on wipe issue.
  return false;

  // macOS and Windows hosts have the same conditions and can be wiped if they
  // are enrolled in MDM and the MDM is enabled.
  const canWipeMacOrWindows =
    (hostPlatform === "darwin" || hostPlatform === "windows") &&
    isFleetMdm &&
    isMdmEnabledAndConfigured &&
    isEnrolledInMdm;

  return (
    isPremiumTier &&
    !isLocked &&
    (hostPlatform === "linux" || canWipeMacOrWindows) &&
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
  isLocked,
  isGlobalAdmin,
  isGlobalMaintainer,
  isGlobalObserver,
  isTeamAdmin,
  isTeamMaintainer,
  isTeamObserver,
  isFleetMdm,
  isEnrolledInMdm,
  isMdmEnabledAndConfigured,
  hostPlatform,
}: IHostActionConfigOptions) => {
  const canLockDarwin =
    hostPlatform === "darwin" &&
    isFleetMdm &&
    isMdmEnabledAndConfigured &&
    isEnrolledInMdm;

  return (
    isPremiumTier &&
    isLocked &&
    (isGlobalAdmin ||
      isGlobalMaintainer ||
      isGlobalObserver ||
      isTeamAdmin ||
      isTeamMaintainer ||
      isTeamObserver) &&
    (canLockDarwin || hostPlatform === "windows" || hostPlatform === "linux")
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
    isLocking,
    isWiping,
    isUnlocking,
    isLocked,
    isWiped,
  }: IHostActionConfigOptions
) => {
  const disableOptions = (optionsToDisable: IDropdownOption[]) => {
    optionsToDisable.forEach((option) => {
      option.disabled = true;
    });
  };

  let optionsToDisable: IDropdownOption[] = [];
  if (!isHostOnline) {
    optionsToDisable = optionsToDisable.concat(
      options.filter(
        (option) => option.value === "query" || option.value === "mdmOff"
      )
    );
  }
  if (isSandboxMode) {
    optionsToDisable = optionsToDisable.concat(
      options.filter((option) => option.value === "transfer")
    );
  }
  if (isLocking || isWiping) {
    optionsToDisable = optionsToDisable.concat(
      options.filter(
        (option) =>
          option.value === "query" ||
          option.value === "mdmOff" ||
          option.value === "lock" ||
          option.value === "wipe"
      )
    );
  }
  if (isLocked || isWiped) {
    optionsToDisable = optionsToDisable.concat(
      options.filter(
        (option) => option.value === "query" || option.value === "mdmOff"
      )
    );
  }
  if (isUnlocking) {
    optionsToDisable = optionsToDisable.concat(
      options.filter((option) => option.value === "unlock")
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

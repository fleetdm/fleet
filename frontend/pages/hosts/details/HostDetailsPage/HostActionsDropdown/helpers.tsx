import React from "react";
import { IDropdownOption } from "interfaces/dropdownOption";
import { cloneDeep } from "lodash";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";

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
  isTeamAdmin: boolean;
  isTeamMaintainer: boolean;
  isHostOnline: boolean;
  isEnrolledInMdm: boolean;
  isFleetMdm: boolean;
  isMdmEnabledAndConfigured: boolean;
  doesStoreEncryptionKey: boolean;
  isSandboxMode: boolean;
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

  return options;
};

const setOptionsAsDisabled = (
  options: IDropdownOption[],
  isHostOnline: boolean,
  isSandboxMode: boolean
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

  options = setOptionsAsDisabled(
    options,
    config.isHostOnline,
    config.isSandboxMode
  );

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

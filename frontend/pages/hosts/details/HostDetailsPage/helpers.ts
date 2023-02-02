import { IDropdownOption } from "interfaces/dropdownOption";

const DEFAULT_OPTIONS: IDropdownOption[] = [
  {
    label: "Transfer",
    value: "transfer",
    disabled: false,
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
];

// eslint-disable-next-line import/prefer-default-export
interface IHostActionConfigOptions {
  isPremiumTier: boolean;
  isGlobalAdmin: boolean;
  isGlobalMaintainer: boolean;
  isTeamAdmin: boolean;
  isTeamMaintainer: boolean;
  isHostOnline: boolean;
  doesStoreEncryptionKey: boolean;
}

const filterOutOptions = (
  options: IDropdownOption[],
  config: IHostActionConfigOptions
) => {
  const {
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    doesStoreEncryptionKey,
  } = config;

  if (!isPremiumTier) {
    options = options.filter(
      (option) =>
        option.value !== "diskEncryption" && option.value !== "transfer"
    );
  }

  if (!doesStoreEncryptionKey) {
    options = options.filter((option) => option.value !== "diskEncryption");
  }

  if (!isGlobalAdmin && !isGlobalMaintainer) {
    options = options.filter((option) => option.value !== "transfer");

    if (!isTeamAdmin && !isTeamMaintainer) {
      options = options.filter(
        (option) =>
          option.value !== "query" &&
          option.value !== "delete" &&
          option.value !== "mdmOff"
      );
    }
  }

  return options;
};

const setOptionsAsDisabled = (
  options: IDropdownOption[],
  isHostOnline: boolean
) => {
  if (!isHostOnline) {
    const disableOptions = options.filter(
      (option) => option.value === "query" || option.value === "mdmOff"
    );
    disableOptions.forEach((option) => {
      option.disabled = true;
    });
  }

  return options;
};

/**
 * Generate the host actions options depending on the configuration. There are
 * many variations of the options that are shown/not shown or disabled/enabled
 * which are all controlled by the configurations options.
 */
// eslint-disable-next-line import/prefer-default-export
export const generateHostActionOptions = (config: IHostActionConfigOptions) => {
  let options = DEFAULT_OPTIONS;
  options = filterOutOptions(options, config);
  options = setOptionsAsDisabled(options, config.isHostOnline);
  return options;
};

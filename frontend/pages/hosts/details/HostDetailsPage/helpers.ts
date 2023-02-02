import { IDropdownOption } from "interfaces/dropdownOption";
import { cloneDeep } from "lodash";

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

  // disk encryption and transfer filtered out if not premium
  if (!isPremiumTier) {
    options = options.filter(
      (option) =>
        option.value !== "diskEncryption" && option.value !== "transfer"
    );
  }

  // disk encryption filtered out if we do not store it
  if (!doesStoreEncryptionKey) {
    options = options.filter((option) => option.value !== "diskEncryption");
  }

  // transfer filtered out if not global admin/maintainer
  if (!isGlobalAdmin && !isGlobalMaintainer) {
    options = options.filter((option) => option.value !== "transfer");

    // query, delete, mdmOff filtered out if not global admin/maintainer or team admin/maintainer
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
  // disabled query and mdmOff if host if offline
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
 * Generates the host actions options depending on the configuration. There are
 * many variations of the options that are shown/not shown or disabled/enabled
 * which are all controlled by the configurations options argument.
 */
// eslint-disable-next-line import/prefer-default-export
export const generateHostActionOptions = (config: IHostActionConfigOptions) => {
  // deep clone to always start with a fresh copy of the default options.
  let options = cloneDeep(DEFAULT_OPTIONS);
  options = filterOutOptions(options, config);
  options = setOptionsAsDisabled(options, config.isHostOnline);
  return options;
};

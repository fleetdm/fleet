import { QueryParams, parseQueryValueToNumberOrUndefined } from "utilities/url";
import stringUtils from "utilities/strings/stringUtils";
import { tooltipTextWithLineBreaks } from "utilities/helpers";

export type ISoftwareDropdownFilterVal =
  | "allSoftware"
  | "installableSoftware"
  | "selfServiceSoftware";

export const SOFTWARE_TITLES_DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "All software",
    value: "allSoftware",
    helpText: "All software installed on your hosts.",
  },
  {
    disabled: false,
    label: "Available for install",
    value: "installableSoftware",
    helpText: "Software that can be installed on your hosts.",
  },
  {
    disabled: false,
    label: "Self-service",
    value: "selfServiceSoftware",
    helpText: "Software that end users can install from Fleet Desktop.",
  },
];

export const buildSoftwareFilterQueryParams = (
  val: ISoftwareDropdownFilterVal
) => {
  switch (val) {
    case "installableSoftware":
      return { availableForInstall: true };
    case "selfServiceSoftware":
      return { selfService: true };
    default:
      return {};
  }
};

// TODO: Consider parsing SoftwarePage query params to change from type string
export const getSoftwareFilterFromQueryParams = (queryParams: QueryParams) => {
  const { available_for_install, self_service } = queryParams;
  switch (true) {
    case available_for_install === "true":
      return "installableSoftware";
    case self_service === "true":
      return "selfServiceSoftware";
    default:
      return "allSoftware";
  }
};

export const isValidNumber = (
  value: any,
  min?: number,
  max?: number
): value is number => {
  // Check if the value is a number and not NaN
  const isNumber = typeof value === "number" && !isNaN(value);

  // If min or max is provided, check if the number is within the range
  const withinRange =
    (min === undefined || value >= min) && (max === undefined || value <= max);

  return isNumber && withinRange;
};

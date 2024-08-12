import { QueryParams } from "utilities/url";

export type ISoftwareDropdownFilterVal =
  | "allSoftware"
  | "vulnerableSoftware"
  | "installableSoftware"
  | "selfServiceSoftware";

export const SOFTWARE_VERSIONS_DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "All software",
    value: "allSoftware",
    helpText: "All software installed on your hosts.",
  },
];

export const SOFTWARE_TITLES_DROPDOWN_OPTIONS = [
  ...SOFTWARE_VERSIONS_DROPDOWN_OPTIONS,
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

export const getSoftwareFilterForQueryKey = (
  val: ISoftwareDropdownFilterVal
) => {
  switch (val) {
    case "installableSoftware":
      return { availableForInstall: true };
    case "selfServiceSoftware":
      return { selfService: true };
    case "vulnerableSoftware":
      return { vulnerable: true };
    default:
      return {};
  }
};

export const getSoftwareFilterFromQueryParams = (queryParams: QueryParams) => {
  const { vulnerable, available_for_install, self_service } = queryParams;
  switch (true) {
    case available_for_install === "true":
      return "installableSoftware";
    case self_service === "true":
      return "selfServiceSoftware";
    case vulnerable === "true":
      return "vulnerableSoftware";
    default:
      return "allSoftware";
  }
};

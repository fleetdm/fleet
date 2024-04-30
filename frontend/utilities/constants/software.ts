export type ISoftwareDropdownFilterVal =
  | "allSoftware"
  | "vulnerableSoftware"
  | "installableSoftware";

export const VULNERABLE_DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "All software",
    value: "allSoftware",
    helpText: "All software installed on your hosts.",
  },
  {
    disabled: false,
    label: "Vulnerable software",
    value: "vulnerableSoftware",
    helpText:
      "All software installed on your hosts with detected vulnerabilities.",
  },
  {
    disabled: false,
    label: "Available for install",
    value: "installableSoftware",
    helpText: "Software that can be installed on your hosts.",
  },
];

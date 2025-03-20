import { IDropdownOption } from "interfaces/dropdownOption";

const DEFAULT_CERT_AUTHORITY_OPTIONS: IDropdownOption[] = [
  { label: "Digicert", value: "digicert" },
  {
    label: "Microsoft NDES (Network Device Enrollment Service)",
    value: "ndes",
  },
  {
    label: "Custom (SCEP: Simple Certificate Enrollment Protocol)",
    value: "custom",
  },
];

export const generateDropdownOptions = (hasNDESCert: boolean) => {
  if (!hasNDESCert) {
    return DEFAULT_CERT_AUTHORITY_OPTIONS;
  }

  const ndesOption = DEFAULT_CERT_AUTHORITY_OPTIONS[1];
  ndesOption.disabled = true;
  ndesOption.tooltipContent = "Only one NDES can be added.";

  return DEFAULT_CERT_AUTHORITY_OPTIONS;
};

const DEFAULT_ERROR_MESSAGE =
  "Couldn't add certificate authority. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown) => {
  return DEFAULT_ERROR_MESSAGE;
};

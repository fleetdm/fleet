import React from "react";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { IDropdownOption } from "interfaces/dropdownOption";
import { getErrorReason } from "interfaces/errors";

import CustomLink from "components/CustomLink";

const DEFAULT_CERT_AUTHORITY_OPTIONS: IDropdownOption[] = [
  { label: "DigiCert", value: "digicert" },
  { label: "Hydrant EST (Enrollment Over Secure Transport)", value: "hydrant" },
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

  // We only allow one NDES configuration, if ones exists disable the option and
  // add a tooltip.
  const ndesOption = DEFAULT_CERT_AUTHORITY_OPTIONS.find((option) => {
    return option.value === "ndes";
  });
  if (ndesOption) {
    ndesOption.disabled = true;
    ndesOption.tooltipContent = "Only one NDES can be added.";
  }

  return DEFAULT_CERT_AUTHORITY_OPTIONS;
};

/**
 * errors used in the add certificate authority flow
 */
const DEFAULT_ERROR = "Please try again.";
const INVALID_API_TOKEN_ERROR =
  "Invalid API token. Please correct and try again.";
const INVALID_PROFILE_GUID_ERROR =
  "Invalid profile GUID. Please correct and try again.";
const INVALID_URL_ERROR = "Invalid URL. Please correct and try again.";
const PRIVATE_KEY_NOT_CONFIGURED_ERROR = (
  <>
    Private key must be configured.{" "}
    <CustomLink
      text="Learn more"
      url={`${LEARN_MORE_ABOUT_BASE_LINK}/fleet-server-private-key`}
      newTab
      variant="flash-message-link"
    />
  </>
);
const INVALID_SCEP_URL_ERROR =
  "Invalid SCEP URL. Please correct and try again.";
const INVALID_ADMIN_URL_OR_CREDENTIALS_ERROR =
  "Invalid admin URL or credentials. Please correct and try again.";
const NDES_PASSWORD_CACHE_FULL_ERROR =
  "The NDES password cache is full. Please increase the number of cached passwords in NDES and try again.";
const INVALID_CHALLENGE_ERROR =
  "Invalid challenge. Please correct and try again.";

/**
 * Gets the error message we want to display from the api error message.
 * This is used in both add and edit certificate authority flows.
 */
export const getDisplayErrMessage = (err: unknown) => {
  let message: string | JSX.Element = DEFAULT_ERROR;
  const reason = getErrorReason(err);

  if (reason.includes("invalid API token")) {
    message = INVALID_API_TOKEN_ERROR;
  } else if (reason.includes("invalid profile GUID")) {
    message = INVALID_PROFILE_GUID_ERROR;
  } else if (
    reason.includes("invalid URL") ||
    reason.includes("no such host")
  ) {
    message = INVALID_URL_ERROR;
  } else if (reason.includes("private key")) {
    message = PRIVATE_KEY_NOT_CONFIGURED_ERROR;
  } else if (reason.includes("invalid SCEP URL")) {
    message = INVALID_SCEP_URL_ERROR;
  } else if (reason.includes("invalid admin URL or credentials")) {
    message = INVALID_ADMIN_URL_OR_CREDENTIALS_ERROR;
  } else if (reason.includes("password cache is full")) {
    message = NDES_PASSWORD_CACHE_FULL_ERROR;
  } else if (reason.includes("invalid challenge")) {
    message = INVALID_CHALLENGE_ERROR;
  } else {
    message = DEFAULT_ERROR;
  }

  return message;
};

export const getErrorMessage = (err: unknown) => {
  return (
    <>Couldn&apos;t add certificate authority. {getDisplayErrMessage(err)}</>
  );
};

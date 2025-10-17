import React from "react";

import { IAddCertAuthorityBody } from "services/entities/certificates";
import { ICertificateAuthorityType } from "interfaces/certificates";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { IDropdownOption } from "interfaces/dropdownOption";
import { getErrorReason } from "interfaces/errors";

import CustomLink from "components/CustomLink";

import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import { INDESFormData } from "../NDESForm/NDESForm";
import { ICustomSCEPFormData } from "../CustomSCEPForm/CustomSCEPForm";
import { IHydrantFormData } from "../HydrantForm/HydrantForm";
import { ISmallstepFormData } from "../SmallstepForm/SmallstepForm";

// FIXME: do we care about the order of these? Should we alphabetize them or something?
const DEFAULT_CERT_AUTHORITY_OPTIONS: IDropdownOption[] = [
  { label: "DigiCert", value: "digicert" },
  {
    label: "Hydrant EST (Enrollment Over Secure Transport)",
    value: "hydrant",
  },
  {
    label: "Microsoft NDES (Network Device Enrollment Service)",
    value: "ndes_scep_proxy",
  },
  {
    label: "Custom SCEP (Simple Certificate Enrollment Protocol)",
    value: "custom_scep_proxy",
  },
  { label: "Smallstep", value: "smallstep" },
];

/**
 * conditionally generates the dropdown options disabling the ndes option
 * if one already exists
 */
export const generateDropdownOptions = (hasNDESCert: boolean) => {
  if (!hasNDESCert) {
    return DEFAULT_CERT_AUTHORITY_OPTIONS;
  }

  // We only allow one NDES configuration, if ones exists disable the option and
  // add a tooltip.
  const ndesOption = DEFAULT_CERT_AUTHORITY_OPTIONS.find((option) => {
    return option.value === "ndes_scep_proxy";
  });
  if (ndesOption) {
    ndesOption.disabled = true;
    ndesOption.tooltipContent = "Only one NDES can be added.";
  }

  return DEFAULT_CERT_AUTHORITY_OPTIONS;
};

/**
 * Generates the data to be sent to the API to add a new certificate authority.
 * This function constructs the request body based on the selected certificate authority type
 * and the provided form data.
 */
// eslint-disable-next-line import/prefer-default-export
export const generateAddCertAuthorityData = (
  certAuthorityType: ICertificateAuthorityType,
  formData: ICertFormData
): IAddCertAuthorityBody | undefined => {
  switch (certAuthorityType) {
    case "ndes_scep_proxy":
      // eslint-disable-next-line no-case-declarations
      const {
        scepURL,
        adminURL,
        username,
        password,
      } = formData as INDESFormData;
      return {
        ndes_scep_proxy: {
          url: scepURL,
          admin_url: adminURL,
          username,
          password,
        },
      };
    case "digicert":
      // eslint-disable-next-line no-case-declarations
      const {
        name,
        url: digicertUrl,
        apiToken,
        profileId,
        commonName,
        userPrincipalName,
        certificateSeatId,
      } = formData as IDigicertFormData;
      return {
        digicert: {
          name,
          url: digicertUrl,
          api_token: apiToken,
          profile_id: profileId,
          certificate_common_name: commonName,
          certificate_user_principal_names: [userPrincipalName],
          certificate_seat_id: certificateSeatId,
        },
      };
    case "custom_scep_proxy":
      // eslint-disable-next-line no-case-declarations
      const {
        name: customSCEPName,
        scepURL: customSCEPUrl,
        challenge,
      } = formData as ICustomSCEPFormData;
      return {
        custom_scep_proxy: {
          name: customSCEPName,
          url: customSCEPUrl,
          challenge,
        },
      };
    case "hydrant":
      // eslint-disable-next-line no-case-declarations
      const {
        name: hydrantName,
        url,
        clientId,
        clientSecret,
      } = formData as IHydrantFormData;
      return {
        hydrant: {
          name: hydrantName,
          url,
          client_id: clientId,
          client_secret: clientSecret,
        },
      };
    case "smallstep":
      // eslint-disable-next-line no-case-declarations
      const {
        name: smallstepName,
        scepURL: smallstepScepURL,
        challengeURL,
        username: smallstepUsername,
        password: smallstepPassword,
      } = formData as ISmallstepFormData;
      return {
        smallstep: {
          name: smallstepName,
          url: smallstepScepURL,
          challenge_url: challengeURL,
          username: smallstepUsername,
          password: smallstepPassword,
        },
      };
    default:
      return undefined;
  }
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
const INVALID_CHALLENGE_URL_OR_CREDENTIALS_ERROR =
  "Invalid challenge URL or credentials. Please correct and try again.";

/**
 * Gets the error message we want to display from the api error message.
 * This is used in both add and edit certificate authority flows.
 */
export const getDisplayErrMessage = (err: unknown): string | JSX.Element => {
  let message: string | JSX.Element = DEFAULT_ERROR;
  const reason = getErrorReason(err).toLowerCase();

  if (reason.includes("invalid api token")) {
    message = INVALID_API_TOKEN_ERROR;
  } else if (reason.includes("invalid profile guid")) {
    message = INVALID_PROFILE_GUID_ERROR;
  } else if (
    reason.includes("invalid url") ||
    reason.includes("no such host")
  ) {
    message = INVALID_URL_ERROR;
  } else if (reason.includes("private key")) {
    message = PRIVATE_KEY_NOT_CONFIGURED_ERROR;
  } else if (reason.includes("invalid scep url")) {
    message = INVALID_SCEP_URL_ERROR;
  } else if (reason.includes("invalid admin url or credentials")) {
    message = INVALID_ADMIN_URL_OR_CREDENTIALS_ERROR;
  } else if (reason.includes("password cache is full")) {
    message = NDES_PASSWORD_CACHE_FULL_ERROR;
  } else if (reason.includes("invalid challenge url")) {
    message = INVALID_CHALLENGE_URL_OR_CREDENTIALS_ERROR;
  } else if (reason.includes("invalid challenge")) {
    message = INVALID_CHALLENGE_ERROR;
  } else {
    message = DEFAULT_ERROR;
  }

  return message;
};

export const getErrorMessage = (err: unknown): JSX.Element => {
  return (
    <>Couldn&apos;t add certificate authority. {getDisplayErrMessage(err)}</>
  );
};

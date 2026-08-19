import React from "react";

import CustomLink from "components/CustomLink";
import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  IHostMdmProfile,
} from "interfaces/mdm";

import { IHostMdmProfileWithAddedStatus } from "../OSSettingsTableConfig";

/**
 * The Fleet Android app reports certificate failures in its own terms, e.g. "Failed to create
 * CSR: ...". Those reach both Host > OS settings and the end user's My Device page, where SCEP
 * and CSR mean nothing and there is nothing the reader can act on. Each known reason is restated
 * in plain language; anything unrecognized falls through to the reported text so a new failure is
 * never swallowed. The raw text stays on the API response either way.
 */
const ANDROID_CERTIFICATE_ERRORS: [prefix: string, message: string][] = [
  [
    "Network error during SCEP enrollment",
    "Fleet couldn't reach the certificate authority.",
  ],
  ["SCEP enrollment failed", "The certificate authority rejected the request."],
  [
    "Certificate validation failed",
    "The host couldn't validate the certificate from the certificate authority.",
  ],
  ["Failed to generate key pair", "The host couldn't generate a private key."],
  [
    "Failed to create CSR",
    "The host couldn't create a certificate signing request.",
  ],
  ["Invalid configuration", "The certificate configuration isn't valid."],
  [
    "Certificate installation failed",
    "The host couldn't install the certificate.",
  ],
];

/** Restates a Fleet Android app certificate error in plain language, or null if unrecognized. */
export const formatAndroidCertificateError = (
  detail: IHostMdmProfile["detail"]
) => {
  const trimmed = detail.trim();
  const match = ANDROID_CERTIFICATE_ERRORS.find(([prefix]) =>
    trimmed.startsWith(prefix)
  );
  return match ? match[1] : null;
};

/** Whether this row is an Android certificate template rather than a configuration profile. */
export const isAndroidCertificateProfile = (
  profile: IHostMdmProfileWithAddedStatus
) => profile.profile_uuid === FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID;

const formatAndroidProfileNotAppliedError = (
  detail: IHostMdmProfile["detail"]
) => {
  if (
    detail.includes("settings couldn't apply to a host") ||
    detail.includes("Other settings are applied")
  ) {
    return (
      <>
        {detail}{" "}
        <CustomLink
          text="Learn more"
          url="https://fleetdm.com/learn-more-about/android-profile-errors"
          newTab
          variant="tooltip-link"
        />
      </>
    );
  }
  return null;
};

/**
 * formatDetailCertificateError generates the formatted detail for certain errors related to
 * certificate profiles. It return a JSX element with the formatted message or null if
 * the detail does not match any of the expected patterns.
 */
const formatDetailCertificateError = (detail: IHostMdmProfile["detail"]) => {
  const formattedCertificatesPath = (
    <b>
      Settings {">"} Integrations {">"} Certificates
    </b>
  );

  const matchTokenErr = detail.match(
    /get certificate from (?:DigiCert|Digicert|digicert).*token configured in (?<ca>.*) certificate authority is invalid/
  );
  if (matchTokenErr?.groups) {
    return (
      <>
        Couldn&apos;t get certificate from DigiCert. The <b>API token</b>{" "}
        configured in <b>{matchTokenErr.groups.ca}</b> certificate authority is
        invalid. Please go to {formattedCertificatesPath}, correct it and
        resend.
      </>
    );
  }

  const matchProfileIdErr = detail.match(
    /get certificate from (?:DigiCert|Digicert|digicert) for (?<ca>.*)\..*POST request: 410.*Profile with id.*was deleted/
  );
  const matchDeletedProfileErr = detail.match(
    /get certificate from (?:DigiCert|Digicert|digicert) for (?<ca>.*)\..*POST request: 400.*deleted or suspended Profile/
  );
  if (matchProfileIdErr?.groups || matchDeletedProfileErr?.groups) {
    return (
      <>
        Couldn&apos;t get certificate from DigiCert. The <b>Profile GUID</b>{" "}
        configured in{" "}
        <b>
          {matchProfileIdErr?.groups?.ca || matchDeletedProfileErr?.groups?.ca}
        </b>{" "}
        certificate authority doesn&apos;t exist. Please go to{" "}
        {formattedCertificatesPath}, correct it and resend.
      </>
    );
  }

  const matchFleetVarErr = detail.match(
    /populate (?<field>.*) because (?<ca>.*) certificate authority does(?:n.t| not) exist/
  );
  if (matchFleetVarErr?.groups) {
    return (
      <>
        Fleet couldn&apos;t populate {matchFleetVarErr.groups.field} because{" "}
        <b>{matchFleetVarErr.groups.ca}</b> certificate authority doesn&apos;t
        exist. Please go to{" "}
        <b>
          Settings {">"} Integrations {">"} Certificates
        </b>
        , add it and resend the configuration profile.
      </>
    );
  }

  return null;
};

/**
 * formatDetailIdpEmailError generates the formatted detail for certain errors related to
 * host IdP email profiles. It returns a JSX element with the formatted message or null if
 * the detail does not match any of the expected patterns.
 */
const formatDetailIdpEmailError = (detail: IHostMdmProfile["detail"]) => {
  if (detail.includes("There is no IdP email for this host.")) {
    return (
      <>
        There is no IdP email for this host.
        <br />
        Fleet couldn&apos;t populate
        <br />
        $FLEET_VAR_HOST_END_USER_EMAIL_IDP.
        <br />
        <CustomLink
          text="Learn more"
          url="https://fleetdm.com/learn-more-about/idp-email"
          newTab
          variant="tooltip-link"
        />
      </>
    );
  }
  return null;
};

/**
 * generates the formatted tooltip for the error column.
 * the expected format of the error string is:
 * "key1: value1, key2: value2, key3: value3"
 */
const formatDetailWindowsProfile = (detail: string) => {
  const keyValuePairs = detail.split(/, */);
  const formattedElements: JSX.Element[] = [];

  // Special case to handle bitlocker and certificate install error messages.
  // They do not follow the expected string format so we will just render the error message as is.
  if (
    detail.includes("BitLocker") ||
    detail.includes("preparing volume for encryption") ||
    detail.startsWith("Couldn't install certificate")
  ) {
    return detail;
  }

  keyValuePairs.forEach((pair, i) => {
    const [key, value] = pair.split(/:(.*)/).map((str) => str.trim());
    if (key && value) {
      formattedElements.push(
        <span key={key}>
          <b>{key}:</b> {value}
          {/* dont add the trailing comma for the last element */}
          {i !== keyValuePairs.length - 1 && (
            <>
              ,<br />
            </>
          )}
        </span>
      );
    }
  });

  return formattedElements.length ? <>{formattedElements}</> : detail;
};

/**
 * generates the error tooltip for the error column. This will be formatted or
 * unformatted.
 */
const generateErrorTooltip = (
  profile: IHostMdmProfileWithAddedStatus
): React.ReactNode => {
  if (profile.status !== "failed" || !profile.detail) return null;

  // Android certificates read the same here as they do while retrying, so an admin doesn't see
  // the failure reworded the moment Fleet stops retrying it.
  if (isAndroidCertificateProfile(profile)) {
    return formatAndroidCertificateError(profile.detail) ?? profile.detail;
  }

  // Special case to handle IdP email errors
  const idpEmailError = formatDetailIdpEmailError(profile.detail);
  if (idpEmailError) {
    return idpEmailError;
  }

  // Special case to handle certificate profile errors
  const certificateError = formatDetailCertificateError(profile.detail);
  if (certificateError) {
    return certificateError;
  }

  const androidProfileNotAppliedError = formatAndroidProfileNotAppliedError(
    profile.detail
  );
  if (androidProfileNotAppliedError) {
    return androidProfileNotAppliedError;
  }

  if (profile.platform === "windows") {
    return formatDetailWindowsProfile(profile.detail);
  }

  return profile.detail;
};

export default generateErrorTooltip;

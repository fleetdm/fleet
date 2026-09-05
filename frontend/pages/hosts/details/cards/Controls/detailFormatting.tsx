import React from "react";

import CustomLink from "components/CustomLink";
import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  IHostMdmProfile,
} from "interfaces/mdm";

import { IHostMdmProfileWithAddedStatus } from "./OSSettingsTableConfig";

/**
 * The Fleet Android app reports certificate failures in its own terms, e.g. "Failed to create
 * CSR: ...". Those reach both Host > Controls and the end user's My device page, where SCEP and
 * CSR mean nothing and there is nothing the reader can act on. Each known reason is restated in
 * plain language; anything unrecognized falls through to the reported text so a new failure is
 * never swallowed. The app's own wording stays in the details block either way.
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

/**
 * When an Android certificate install fails, Fleet resets the certificate and delivers it again,
 * so the control goes back to an in-progress status and is otherwise indistinguishable from a
 * first delivery. The server decides this: a manual resend leaves the same retry count behind, and
 * telling the two apart needs state that does not survive into the API response.
 */
export const isRetryingAndroidCertificate = (
  profile?: IHostMdmProfileWithAddedStatus
) => !!profile && isAndroidCertificateProfile(profile) && !!profile.retrying;

/**
 * Builds the details message for a retrying Android certificate, e.g. "Fleet couldn't reach the
 * certificate authority. Retrying enrollment (attempt 2 of 4)." The numbers are labelled as
 * attempts because they count the initial attempt, and so run one higher than the
 * retry_count/max_retries the API reports. The host does not have to report an error message with
 * a failure, in which case the retry stands on its own.
 */
export const getAndroidCertificateRetryMessage = (
  profile: IHostMdmProfileWithAddedStatus
) => {
  const attempts =
    profile.retry_count === undefined || profile.max_retries === undefined
      ? ""
      : ` (attempt ${profile.retry_count + 1} of ${profile.max_retries + 1})`;
  const retrying = `Retrying enrollment${attempts}.`;

  const reported = profile.detail?.trim();
  if (!reported) {
    return retrying;
  }

  // Prefer plain language over the app's own wording, falling back to what it reported so an
  // unrecognized failure still reaches the reader.
  const detail = formatAndroidCertificateError(reported) ?? reported;
  const sentence = /[.!?]$/.test(detail) ? detail : `${detail}.`;
  return `${sentence} ${retrying}`;
};

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
        There is no IdP email for this host. Fleet couldn&apos;t populate
        $FLEET_VAR_HOST_END_USER_EMAIL_IDP.{" "}
        <CustomLink
          text="Learn more"
          url="https://fleetdm.com/learn-more-about/idp-email"
          newTab
        />
      </>
    );
  }
  return null;
};

export interface IDetailGuidance {
  message: React.ReactNode;
  /** True when the message already says everything the raw detail says, so the
   * copyable block would repeat it. Lines up with the "Learn more" messages,
   * which quote the detail; the certificate rewrites drop diagnostic text. */
  supersedesDetail: boolean;
}

/** The rewritten, actionable version of a recognized error. Null when the
 * detail matches no known pattern, leaving the raw detail to stand alone. */
export const getDetailGuidance = (
  profile: IHostMdmProfileWithAddedStatus
): IDetailGuidance | null => {
  if (profile.status !== "failed" || !profile.detail) return null;

  // Android certificates read the same here as they do while retrying, so an admin doesn't see
  // the failure reworded the moment Fleet stops retrying it. The raw detail is left in place
  // below, since the rewrite drops the app's diagnostic text.
  if (isAndroidCertificateProfile(profile)) {
    const message = formatAndroidCertificateError(profile.detail);
    return message ? { message, supersedesDetail: false } : null;
  }

  const idpEmailError = formatDetailIdpEmailError(profile.detail);
  if (idpEmailError) {
    return { message: idpEmailError, supersedesDetail: true };
  }

  const certificateError = formatDetailCertificateError(profile.detail);
  if (certificateError) {
    return { message: certificateError, supersedesDetail: false };
  }

  const androidError = formatAndroidProfileNotAppliedError(profile.detail);
  if (androidError) {
    return { message: androidError, supersedesDetail: true };
  }

  return null;
};

/** Windows details arrive as one "key: value, key: value" line. A CSP profile
 * can carry hundreds of URI paths, so split it one result per line. */
const splitWindowsDetailLines = (detail: string) => {
  // BitLocker and certificate install errors are prose, not key/value pairs,
  // and splitting them on commas mangles the sentence.
  if (
    detail.includes("BitLocker") ||
    detail.includes("preparing volume for encryption") ||
    detail.startsWith("Couldn't install certificate")
  ) {
    return detail;
  }

  const lines = detail
    .split(/, */)
    .filter((pair) => /^[^:]+:(.*)$/.test(pair.trim()));

  return lines.length ? lines.join("\n") : detail;
};

/** The raw detail as plain text, for the output block and the clipboard. */
export const getDetailText = (
  profile: IHostMdmProfileWithAddedStatus
): string => {
  if (!profile.detail) return "";

  if (profile.platform === "windows") {
    return splitWindowsDetailLines(profile.detail);
  }

  return profile.detail;
};

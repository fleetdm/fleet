import React from "react";

import CustomLink from "components/CustomLink";
import { IHostMdmProfile } from "interfaces/mdm";

import { IHostMdmProfileWithAddedStatus } from "./OSSettingsTableConfig";

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
  /**
   * True when the message already says everything the raw detail says, so the
   * copyable block below it would only repeat itself.
   *
   * Today this lines up exactly with the messages carrying a "Learn more" link:
   * those quote the detail and append a doc link, while the certificate
   * rewrites drop diagnostic text (status codes, profile IDs) that's still
   * worth having.
   */
  supersedesDetail: boolean;
}

/**
 * getDetailGuidance returns the rewritten, actionable version of an error the
 * raw detail can't convey on its own — a misconfigured certificate authority, a
 * missing IdP email, an Android setting Fleet can't apply. Returns null when the
 * detail doesn't match a known pattern, in which case the raw detail stands on
 * its own.
 */
export const getDetailGuidance = (
  profile: IHostMdmProfileWithAddedStatus
): IDetailGuidance | null => {
  if (profile.status !== "failed" || !profile.detail) return null;

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

/**
 * Windows details arrive as a single "key1: value1, key2: value2" line. A CSP
 * profile can carry hundreds of URI paths that way, so break it into one result
 * per line — the whole point of this view is finding the failures in that list.
 */
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

/**
 * getDetailText returns the control's raw detail as plain text, ready for both
 * the copyable output block and the clipboard.
 */
export const getDetailText = (
  profile: IHostMdmProfileWithAddedStatus
): string => {
  if (!profile.detail) return "";

  if (profile.platform === "windows") {
    return splitWindowsDetailLines(profile.detail);
  }

  return profile.detail;
};

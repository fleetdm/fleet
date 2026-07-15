import React from "react";

export const ADD_SOFTWARE_ERROR_PREFIX = "Couldn't add.";
export const DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE = `${ADD_SOFTWARE_ERROR_PREFIX} Please try again.`;
export const REQUEST_TIMEOUT_ERROR_MESSAGE = `${ADD_SOFTWARE_ERROR_PREFIX} Request timeout. Please make sure your server and load balancer timeout is long enough.`;

export const DIFFERENT_FILE_TYPE_MESSAGE =
  "The selected package is for a different file type.";

/**
 * Ensures that a string ends with a period.
 * If the string is empty or already ends with a period, it is returned unchanged.
 */
export const ensurePeriod = (str: string) => {
  if (str && !str.endsWith(".")) {
    return `${str}.`;
  }
  return str;
};

/**
 * Renders backend "already has ..." conflicts as flash-message JSX, bolding
 * the software title and fleet name per design. Returns null if the reason
 * doesn't match a known pattern so callers can fall back to generic handling.
 */
export const formatAlreadyAvailableInstallMessage = (msg: string) => {
  // Strip the legacy "Couldn't add software." prefix if present.
  const cleaned = msg.replace(/^Couldn't add software\.?\s*/, "");

  const fmaMatch = cleaned.match(
    /^(.+?) already has a Fleet-maintained app on the (.+?) fleet\./
  );
  if (fmaMatch) {
    return (
      <>
        {ADD_SOFTWARE_ERROR_PREFIX} <b>{fmaMatch[1]}</b> already has a
        Fleet-maintained app on the <b>{fmaMatch[2]}</b> fleet.
      </>
    );
  }

  const vppMatch = cleaned.match(
    /^(.+?) already has an Apple App Store \(VPP\) on the (.+?) fleet\./
  );
  if (vppMatch) {
    return (
      <>
        {ADD_SOFTWARE_ERROR_PREFIX} <b>{vppMatch[1]}</b> already has an Apple
        App Store (VPP) on the <b>{vppMatch[2]}</b> fleet.
      </>
    );
  }

  const packageMatch = cleaned.match(
    /^(.+?) already has a software package on the (.+?) fleet\./
  );
  if (packageMatch) {
    return (
      <>
        {ADD_SOFTWARE_ERROR_PREFIX} <b>{packageMatch[1]}</b> already has a
        software package on the <b>{packageMatch[2]}</b> fleet.
      </>
    );
  }

  const limitMatch = cleaned.match(
    /^(.+?) already has (\d+) packages\. Before adding, delete one you no longer use\./
  );
  if (limitMatch) {
    return (
      <>
        {ADD_SOFTWARE_ERROR_PREFIX} <b>{limitMatch[1]}</b> already has{" "}
        {limitMatch[2]} packages. Before adding, delete one you no longer use.
      </>
    );
  }

  // Legacy generic conflict — kept as a fallback in case a code path still
  // emits it. Matches "<title> already has an installer available for the
  // <fleet> fleet."
  const legacyInstallerMatch = cleaned.match(
    /^(.+?) already has an installer available for the (.+?) fleet\./
  );
  if (legacyInstallerMatch) {
    return (
      <>
        {ADD_SOFTWARE_ERROR_PREFIX} <b>{legacyInstallerMatch[1]}</b> already has
        an installer available for the <b>{legacyInstallerMatch[2]}</b> fleet.
      </>
    );
  }

  // Legacy quote-style: `SoftwareInstaller "X" already exists with fleet "Y".`
  // or `In-house app "X" already exists with fleet "Y".` (emitted by
  // `alreadyExists(...).WithTeamName(...)` in the mysql layer).
  const legacyQuotedMatch = cleaned.match(
    /^(?:SoftwareInstaller|In-house app) "(.+?)" already.+ fleet "(.+?)"\./
  );
  if (legacyQuotedMatch) {
    return (
      <>
        {ADD_SOFTWARE_ERROR_PREFIX} <b>{legacyQuotedMatch[1]}</b> already has an
        installer available for the <b>{legacyQuotedMatch[2]}</b> fleet.
      </>
    );
  }

  return null;
};

/**
 * Format the backend "different file type" error using the software title
 * from the calling flow's context. Returns null if the reason doesn't match
 * or no title is provided so callers can fall back.
 */
export const formatDifferentFileTypeMessage = (
  msg: string,
  softwareTitle?: string
) => {
  if (!softwareTitle || !msg.includes(DIFFERENT_FILE_TYPE_MESSAGE)) {
    return null;
  }
  return (
    <>
      {ADD_SOFTWARE_ERROR_PREFIX} <b>{softwareTitle}</b> already has an
      installer of a different file type.
    </>
  );
};

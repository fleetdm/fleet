import React from "react";

export const ADD_SOFTWARE_ERROR_PREFIX = "Couldn't add.";
export const DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE = `${ADD_SOFTWARE_ERROR_PREFIX} Please try again.`;
export const REQUEST_TIMEOUT_ERROR_MESSAGE = `${ADD_SOFTWARE_ERROR_PREFIX} Request timeout. Please make sure your server and load balancer timeout is long enough.`;

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
 * Matches API messages indicating that a software package or VPP is already available
 * for install on a team.
 *
 * Example matches:
 * - When adding a VPP and the team already has a software package
 * - When adding a software package and the team already has a VPP
 *
 * Returns a formatted React element if matched; otherwise, returns null.
 */
export const formatAlreadyAvailableInstallMessage = (msg: string) => {
  const regex = new RegExp(
    `${ADD_SOFTWARE_ERROR_PREFIX} (.+) already.+on the (.+) team.`
  );
  const match = msg.match(regex);

  if (match) {
    return (
      <>
        {ADD_SOFTWARE_ERROR_PREFIX} <b>{match[1]}</b> already has a package or
        app available for install on the <b>{match[2]}</b> team.{" "}
      </>
    );
  }
  return null;
};

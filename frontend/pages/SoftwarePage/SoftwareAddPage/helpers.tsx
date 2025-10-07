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
 *  Matches API messages after the fixed Couldn't add software. prefix,
 * and renders just the part about what is available for install.
 * Returns a formatted React element if matched; otherwise, returns null. */
export const formatAlreadyAvailableInstallMessage = (msg: string) => {
  // Remove prefix (with or without trailing space)
  const cleaned = msg.replace(/^Couldn't add software\.?\s*/, "");
  // New regex for "<package> already has a package or app available for install on the <team> team."
  const regex = /^(.+?) already.+on the (.+?) team\./;
  const match = cleaned.match(regex);

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

import React from "react";
import { getErrorReason } from "interfaces/errors";
import { trimEnd, upperFirst } from "lodash";

const INSTALL_SOFTWARE_ERROR_PREFIX = "Couldn't install.";
const DEFAULT_ERROR_MESSAGE = `${INSTALL_SOFTWARE_ERROR_PREFIX} Please try again.`;

const createOnlyInstallableOnMacOSMessage = (reason: string) =>
  `Couldn't install. ${reason.replace("darwin", "macOS")}.`;

const createVPPTokenExpiredMessage = () => (
  <>
    {INSTALL_SOFTWARE_ERROR_PREFIX} VPP token expired. Go to{" "}
    <b>
      Settings {">"} Integration {">"} Volume Purchasing Program
    </b>{" "}
    and renew token.
  </>
);

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown) => {
  const reason = upperFirst(trimEnd(getErrorReason(e), "."));

  if (reason.includes("fleetd installed")) {
    return `${INSTALL_SOFTWARE_ERROR_PREFIX}. ${reason}.`;
  } else if (reason.includes("can be installed only on")) {
    return createOnlyInstallableOnMacOSMessage(reason);
  } else if (reason.includes("VPP token expired")) {
    createVPPTokenExpiredMessage();
  } else if (reason.includes("MDM is turned off")) {
    return reason;
  }

  return DEFAULT_ERROR_MESSAGE;
};

import React from "react";
import { getErrorReason } from "interfaces/errors";
import { IHostSoftware, IVPPHostSoftware } from "interfaces/software";
import { trimEnd, upperFirst } from "lodash";

const INSTALL_SOFTWARE_ERROR_PREFIX = "Couldn't install.";
const DEFAULT_INSTALL_ERROR_MESSAGE = `${INSTALL_SOFTWARE_ERROR_PREFIX} Please try again.`;

const UNINSTALL_SOFTWARE_ERROR_PREFIX = "Couldn't uninstall.";
const DEFAULT_UNINSTALL_ERROR_MESSAGE = `${UNINSTALL_SOFTWARE_ERROR_PREFIX} Please try again.`;

export type IHostSWLibraryDropdownFilterVal = "available" | "selfService";

export const DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "All available",
    value: "available",
    helpText: "Software that can be installed on this host.",
  },
  {
    disabled: false,
    label: "Self-service",
    value: "selfService",
    helpText: (
      <>
        Software that end users can install from <b>Fleet Desktop</b> {">"}
        <b>Self-service</b>.
      </>
    ),
  },
];

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

// determines if we want to show the API message as is to the user.
const showAPIMessage = (message: string) => {
  return (
    message.includes("MDM is turned off") ||
    message.includes("No available licenses") ||
    message.includes("Software title is not available for install")
  );
};

export const getInstallErrorMessage = (e: unknown) => {
  const reason = upperFirst(trimEnd(getErrorReason(e), "."));

  if (reason.includes("fleetd installed")) {
    return `${INSTALL_SOFTWARE_ERROR_PREFIX}. ${reason}.`;
  } else if (reason.includes("can be installed only on")) {
    return createOnlyInstallableOnMacOSMessage(reason);
  } else if (reason.includes("VPP token expired")) {
    return createVPPTokenExpiredMessage();
  } else if (showAPIMessage(reason)) {
    return reason;
  }

  return DEFAULT_INSTALL_ERROR_MESSAGE;
};

export const getUninstallErrorMessage = (e: unknown) => {
  const reason = upperFirst(trimEnd(getErrorReason(e), "."));

  if (
    reason.includes("run script") ||
    reason.includes("running script") ||
    reason.includes("have fleetd") ||
    reason.includes("only on")
  ) {
    return `${UNINSTALL_SOFTWARE_ERROR_PREFIX} ${reason}.`;
  } else if (reason.startsWith("Couldn't uninstall software.")) {
    return reason.replace(
      "Couldn't uninstall software.",
      "Couldn't uninstall."
    );
  } else if (reason.startsWith("No uninstall script exists")) {
    return `${UNINSTALL_SOFTWARE_ERROR_PREFIX}. An uninstall script does not exist for this package.`;
  }

  return DEFAULT_UNINSTALL_ERROR_MESSAGE;
};

export const getLastInstall = (software: IHostSoftware | IVPPHostSoftware) =>
  software.software_package?.last_install ||
  software.app_store_app?.last_install ||
  null;

export const getLastUninstall = (software: IHostSoftware) =>
  software.software_package?.last_uninstall || null;

import { IconNames } from "components/icons";
import {
  SoftwareInstallDetailsStatus,
  EnhancedSoftwareInstallUninstallStatus,
  SoftwareInstallStatus,
} from "interfaces/software";

// Install/Uninstall helpers

export const INSTALL_DETAILS_STATUS_ICONS: Record<
  SoftwareInstallDetailsStatus,
  IconNames
> = {
  pending_install: "pending-outline",
  installed: "success",
  uninstalled: "success",
  failed_install: "error",
  pending_uninstall: "pending-outline",
  failed_uninstall: "error",
  // Same "!" glyph as a failure, but the call site renders it muted grey
  // (ui-fleet-black-50): a skip is deferred (app was open), not an error.
  skipped_install: "error-outline",
} as const;

export const SKIPPED_INSTALL_DETAILS_PREFIX =
  "The app was open. It will update once the user closes it and the ";
export const SKIPPED_INSTALL_DETAILS_LINK_TEXT = "policy runs again";
export const SKIPPED_INSTALL_DETAILS_LINK_URL =
  "https://fleetdm.com/learn-more-about/policy-automations";

export const SKIPPED_INSTALL_DETAILS = `${SKIPPED_INSTALL_DETAILS_PREFIX}${SKIPPED_INSTALL_DETAILS_LINK_TEXT}.`;

// A skip stores an empty pre-install query output, so there is nothing to echo back.
export const SKIPPED_PRE_INSTALL_OUTPUT =
  "Query didn't return result or failed\nThe app was open";

const INSTALL_DETAILS_STATUS_PREDICATES: Record<
  EnhancedSoftwareInstallUninstallStatus | "skipped_install",
  string
> = {
  pending_install: "is installing or will install",
  installed: "installed",
  uninstalled: "uninstalled",
  failed_install: "failed to install",
  pending_uninstall: "is uninstalling or will uninstall",
  failed_uninstall: "failed to uninstall",
  pending_script: "is running or will run",
  failed_script: "failed to run",
  ran_script: "ran",
  skipped_install: "skipped install of",
} as const;

export const getInstallDetailsStatusPredicate = (
  status: string | undefined
) => {
  if (!status) {
    return INSTALL_DETAILS_STATUS_PREDICATES.pending_install;
  }
  return (
    INSTALL_DETAILS_STATUS_PREDICATES[
      status.toLowerCase() as EnhancedSoftwareInstallUninstallStatus
    ] || INSTALL_DETAILS_STATUS_PREDICATES.pending_install
  );
};

// Script helpers
export const SCRIPT_DETAILS_STATUS_ICONS: Record<
  SoftwareInstallStatus,
  IconNames
> = {
  pending_install: "pending-outline",
  installed: "success",
  failed_install: "error",
} as const;

const SCRIPT_DETAILS_STATUS_PREDICATES: Record<
  SoftwareInstallStatus,
  string
> = {
  pending_install: "is running or will run",
  installed: "ran",
  failed_install: "failed to run",
} as const;

export const getScriptDetailsStatusPredicate = (status: string | undefined) => {
  if (!status) {
    return SCRIPT_DETAILS_STATUS_PREDICATES.pending_install;
  }
  return (
    SCRIPT_DETAILS_STATUS_PREDICATES[
      status.toLowerCase() as SoftwareInstallStatus
    ] || SCRIPT_DETAILS_STATUS_PREDICATES.pending_install
  );
};

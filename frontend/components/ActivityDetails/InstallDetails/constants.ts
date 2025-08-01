import { IconNames } from "components/icons";
import {
  SoftwareInstallStatus,
  SoftwareUninstallStatus,
} from "interfaces/software";

export const INSTALL_DETAILS_STATUS_ICONS: Record<
  SoftwareInstallStatus | SoftwareUninstallStatus, // former is superset of latter, latter included in union for type system
  IconNames
> = {
  pending_install: "pending-outline",
  installed: "success",
  uninstalled: "success",
  failed_install: "error",
  pending_uninstall: "pending-outline",
  failed_uninstall: "error",
} as const;

const INSTALL_DETAILS_STATUS_PREDICATES: Record<
  SoftwareInstallStatus | SoftwareUninstallStatus, // former is superset of latter, latter included in union for type system
  string
> = {
  pending_install: "is installing or will install",
  installed: "installed",
  uninstalled: "uninstalled",
  failed_install: "failed to install",
  pending_uninstall: "is uninstalling or will uninstall",
  failed_uninstall: "failed to uninstall",
} as const;

export const getInstallDetailsStatusPredicate = (
  status: string | undefined
) => {
  if (!status) {
    return INSTALL_DETAILS_STATUS_PREDICATES.pending_install;
  }
  return (
    INSTALL_DETAILS_STATUS_PREDICATES[
      status.toLowerCase() as SoftwareInstallStatus
    ] || INSTALL_DETAILS_STATUS_PREDICATES.pending_install
  );
};

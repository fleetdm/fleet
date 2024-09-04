import { IconNames } from "components/icons";
import { SoftwareInstallStatus } from "interfaces/software";

export const INSTALL_DETAILS_STATUS_ICONS: Record<
  SoftwareInstallStatus,
  IconNames
> = {
  pending: "pending-outline",
  pending_install: "pending-outline",
  installed: "success-outline",
  failed: "error-outline",
  failed_install: "error-outline",
} as const;

const INSTALL_DETAILS_STATUS_PREDICATES: Record<
  SoftwareInstallStatus,
  string
> = {
  pending: "is installing or will install",
  pending_install: "is installing or will install",
  installed: "installed",
  failed: "failed to install",
  failed_install: "failed to install",
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

export const SOFTWARE_INSTALL_OUTPUT_DISPLAY_LABELS = {
  pre_install_query_output: "Pre-install condition",
  output: "Software install output",
  post_install_script_output: "Post-install script output",
} as const;

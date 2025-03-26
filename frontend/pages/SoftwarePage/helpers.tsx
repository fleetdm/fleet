/**
 * TODO: Major restructure of directories
 * 1. Create /software parent directory similar to /hosts, /queries, /policies...
 * 2. software/SoftwarePage should be its own subdirectory that is only for the main software nav
 * 3. Separate this file into /software/helpers and software/SoftwarePage/helpers
 * 4. software/SoftwarePage will include its child tabs: /SoftwareTitles /SoftwareOS and /SoftwareVulnerabilities
 * 5. Create software/components for components shared across software pages such as SoftwareVppForm and FleetAppDetailsForm
 */

import React from "react";

import { getErrorReason } from "interfaces/errors";
import { ISoftwarePackage, IAppStoreApp } from "interfaces/software";
import { IDropdownOption } from "interfaces/dropdownOption";

/**
 * helper function to generate error message for secret variables based
 * on the error reason.
 */
// eslint-disable-next-line import/prefer-default-export
export const generateSecretErrMsg = (err: unknown) => {
  const reason = getErrorReason(err);

  let errorType = "";
  if (getErrorReason(err, { nameEquals: "install script" })) {
    errorType = "install script";
  } else if (getErrorReason(err, { nameEquals: "post-install script" })) {
    errorType = "post-install script";
  } else if (getErrorReason(err, { nameEquals: "uninstall script" })) {
    errorType = "uninstall script";
  } else if (getErrorReason(err, { nameEquals: "profile" })) {
    errorType = "profile";
  }

  if (errorType === "profile") {
    // for profiles we can get two different error messages. One contains a colon
    // and the other doesn't. We need to handle both cases.
    const message = reason.split(":").pop() ?? "";

    return message
      .replace(/Secret variables?/i, "Variable")
      .replace("missing from database", "doesn't exist.");
  }

  // all other specific error types
  if (errorType) {
    return reason
      .replace(/Secret variables?/i, `Variable used in ${errorType} `)
      .replace("missing from database", "doesn't exist.");
  }

  // no special error type. return generic secret error message
  return reason
    .replace(/Secret variables?/i, "Variable")
    .replace("missing from database", "doesn't exist.");
};

/** Corresponds to automatic_install_policies  */
export type InstallType = "manual" | "automatic";

export const getInstallType = (
  softwarePackage: ISoftwarePackage
): InstallType => {
  return softwarePackage.automatic_install_policies ? "automatic" : "manual";
};

// Used in EditSoftwareModal and PackageForm
export const getTargetType = (
  softwareInstaller: ISoftwarePackage | IAppStoreApp
) => {
  if (!softwareInstaller) return "All hosts";

  return !softwareInstaller.labels_include_any &&
    !softwareInstaller.labels_exclude_any
    ? "All hosts"
    : "Custom";
};

// Used in EditSoftwareModal and PackageForm
export const getCustomTarget = (
  softwareInstaller: ISoftwarePackage | IAppStoreApp
) => {
  if (!softwareInstaller) return "labelsIncludeAny";

  return softwareInstaller.labels_include_any
    ? "labelsIncludeAny"
    : "labelsExcludeAny";
};

// Used in EditSoftwareModal and PackageForm
export const generateSelectedLabels = (
  softwareInstaller: ISoftwarePackage | IAppStoreApp
) => {
  if (
    !softwareInstaller ||
    (!softwareInstaller.labels_include_any &&
      !softwareInstaller.labels_exclude_any)
  ) {
    return {};
  }

  const customTypeKey = softwareInstaller.labels_include_any
    ? "labels_include_any"
    : "labels_exclude_any";

  return (
    softwareInstaller[customTypeKey]?.reduce<Record<string, boolean>>(
      (acc, label) => {
        acc[label.name] = true;
        return acc;
      },
      {}
    ) ?? {}
  );
};

// Used in FleetAppDetailsForm and PackageForm
export const generateHelpText = (
  automaticInstall: boolean,
  customTarget: string
) => {
  if (customTarget === "labelsIncludeAny") {
    return !automaticInstall ? (
      <>
        Software will only be available for install on hosts that{" "}
        <b>have any</b> of these labels:
      </>
    ) : (
      <>
        Software will only be installed on hosts that <b>have any</b> of these
        labels:
      </>
    );
  }

  // this is the case for labelsExcludeAny
  return !automaticInstall ? (
    <>
      Software will only be available for install on hosts that{" "}
      <b>don&apos;t have any</b> of these labels:
    </>
  ) : (
    <>
      Software will only be installed on hosts that <b>don&apos;t have any</b>{" "}
      of these labels:{" "}
    </>
  );
};

// Used in FleetAppDetailsForm and PackageForm
export const CUSTOM_TARGET_OPTIONS: IDropdownOption[] = [
  {
    value: "labelsIncludeAny",
    label: "Include any",
    disabled: false,
  },
  {
    value: "labelsExcludeAny",
    label: "Exclude any",
    disabled: false,
  },
];

export const SELF_SERVICE_TOOLTIP = (
  <>
    End users can install from <b>Fleet Desktop</b> &gt; <b>Self-service</b>.
  </>
);

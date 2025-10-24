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
import {
  IHostSoftware,
  ISoftwarePackage,
  IAppStoreApp,
  ISoftwareTitle,
} from "interfaces/software";
import { IDropdownOption } from "interfaces/dropdownOption";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";

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

export const getSelfServiceTooltip = (isIosOrIpadosApp: boolean) => {
  return isIosOrIpadosApp ? (
    <>End users can install from self-service web app.</>
  ) : (
    <>
      End users can install from <br />
      <strong>Fleet Desktop</strong> &gt; <strong>Self-service</strong>. <br />
      <CustomLink
        newTab
        text="Learn more"
        variant="tooltip-link"
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/self-service-software`}
      />
    </>
  );
};

export const getSelfServiceHelpText = (isIosOrIpadosApp: boolean) => {
  return isIosOrIpadosApp ? (
    <>
      End users can install from self-service.{" "}
      <CustomLink
        newTab
        text="Learn how to deploy self-service"
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/deploy-self-service-to-ios`}
      />
    </>
  ) : (
    <>
      End users can install from <br />
      <strong>Fleet Desktop</strong> &gt; <strong>Self-service</strong>.{" "}
      <CustomLink
        newTab
        text="Learn more"
        variant="tooltip-link"
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/self-service-software`}
      />
    </>
  );
};

export const getAutomaticInstallPoliciesCount = (
  softwareTitle: ISoftwareTitle | IHostSoftware
): number => {
  const { software_package, app_store_app } = softwareTitle;
  if (software_package) {
    return software_package.automatic_install_policies?.length || 0;
  } else if (app_store_app) {
    return app_store_app.automatic_install_policies?.length || 0;
  }
  return 0;
};

// Helper to check safe image src
// Used in SoftwareDetailsSummary in the EditIconModal
export const isSafeImagePreviewUrl = (url?: string | null) => {
  if (typeof url !== "string" || !url) return false;
  try {
    const parsed = new URL(url, window.location.origin);
    // Allow only blob:, data: (for images), or https/http
    if (
      parsed.protocol === "blob:" ||
      parsed.protocol === "data:" ||
      parsed.protocol === "https:" ||
      parsed.protocol === "http:"
    ) {
      // Optionally, for data: URLs, ensure it's an image mime
      if (parsed.protocol === "data:" && !/^data:image\/png/.test(url)) {
        return false;
      }
      return true;
    }
    return false;
  } catch {
    return false;
  }
};

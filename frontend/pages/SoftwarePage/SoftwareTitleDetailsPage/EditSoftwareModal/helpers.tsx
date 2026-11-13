import { isAxiosError } from "axios";
import React from "react";

import { getErrorReason } from "interfaces/errors";
import { IAppStoreApp, ISoftwarePackage } from "interfaces/software";
import {
  generateSecretErrMsg,
  getDisplayedSoftwareName,
} from "pages/SoftwarePage/helpers";
import { ensurePeriod } from "pages/SoftwarePage/SoftwareAddPage/helpers";

export const EDIT_SOFTWARE_ERROR_PREFIX = "Couldn't edit software.";
const DEFAULT_ERROR_MESSAGE = `${EDIT_SOFTWARE_ERROR_PREFIX} Please try again.`;

// Fields the backend cancels pending installs/uninstalls for live in
// `fieldsShouldSideEffect` in ee/server/service/software_installers.go. Self
// service and categories aren't there, so edits that only change those skip
// the "this will cancel pending installs" confirm modal.
const NON_SIDE_EFFECT_KEYS = new Set(["selfService", "categories"]);

export const isNonSideEffectOnlyChange = (updates: Record<string, unknown>) => {
  const keys = Object.keys(updates);
  return keys.length > 0 && keys.every((k) => NON_SIDE_EFFECT_KEYS.has(k));
};

export const getErrorMessage = (
  err: unknown,
  software: ISoftwarePackage | IAppStoreApp
) => {
  const isTimeout =
    isAxiosError(err) &&
    (err.response?.status === 504 || err.response?.status === 408);
  const reason = getErrorReason(err);

  if (isTimeout) {
    return `${EDIT_SOFTWARE_ERROR_PREFIX} Request timeout. Please make sure your server and load balancer timeout is long enough.`;
  } else if (reason.includes("selected package is")) {
    return (
      <>
        Couldn&apos;t edit{" "}
        <b>{getDisplayedSoftwareName(software.name, software.display_name)}</b>.{" "}
        {reason}
      </>
    );
  } else if (reason.includes("Secret variable")) {
    return generateSecretErrMsg(err).replace("Couldn't add", "Couldn't edit");
  } else if (reason.includes("some or all of the categories provided")) {
    return (
      <>
        Couldn&apos;t edit{" "}
        <b>{getDisplayedSoftwareName(software.name, software.display_name)}</b>.{" "}
        Some or all of the categories provided were not found. Please refresh
        the page and try again.
      </>
    );
  }

  if (!reason) {
    return DEFAULT_ERROR_MESSAGE;
  }
  // The edit modal always leads with the product-approved verb. Shared
  // validators now return action-neutral reasons, but some backend messages
  // still carry their own leading verb (e.g. "Couldn't edit.", "Couldn't
  // update."); strip it so the UI shows a single, consistent "Couldn't edit
  // software." rather than the backend's wording or a doubled verb.
  const withoutLeadingVerb = reason.replace(/^Couldn't [^.]*\.\s*/, "");
  return withoutLeadingVerb
    ? `${EDIT_SOFTWARE_ERROR_PREFIX} ${ensurePeriod(withoutLeadingVerb)}`
    : DEFAULT_ERROR_MESSAGE;
};

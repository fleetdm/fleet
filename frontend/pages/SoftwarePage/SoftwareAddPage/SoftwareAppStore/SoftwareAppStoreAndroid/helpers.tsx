import React, { ReactElement } from "react";

import { getErrorReason } from "interfaces/errors";

import {
  ADD_SOFTWARE_ERROR_PREFIX,
  DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE,
  ensurePeriod,
  formatAlreadyAvailableInstallMessage,
} from "../../helpers";

const ANDROID_RECONNECT_PATH =
  "Settings > Integrations > Mobile device management";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown): string | ReactElement => {
  const reason = getErrorReason(e);

  // Dead Android Enterprise: bold the nav path so admins can spot it in a long toast.
  if (reason.includes(ANDROID_RECONNECT_PATH)) {
    const [before, after] = reason.split(ANDROID_RECONNECT_PATH);
    return (
      <>
        {before}
        <strong>{ANDROID_RECONNECT_PATH}</strong>
        {after}
      </>
    );
  }

  // Backend's Android managed-Play copy already starts with "Couldn't add software." —
  // pass through so we don't double-prefix.
  if (
    reason.includes(
      "isn't available in your organization's managed Google Play"
    )
  ) {
    return reason;
  }

  // software is already available for install
  if (reason.toLowerCase().includes("already")) {
    const alreadyAvailableMessage = formatAlreadyAvailableInstallMessage(
      reason
    );
    if (alreadyAvailableMessage) {
      return alreadyAvailableMessage;
    }

    // TODO: What is the check for a Android app? assuming "VPPApp" check won't suffice
    if (reason.includes("VPPApp")) {
      return `${ADD_SOFTWARE_ERROR_PREFIX} The software is already available to install in this fleet.`;
    }
  }

  if (reason) {
    return `${ADD_SOFTWARE_ERROR_PREFIX} ${ensurePeriod(reason)}`;
  }

  return DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE;
};

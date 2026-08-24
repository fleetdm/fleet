import { isAxiosError } from "axios";
import { getErrorReason, hasStatusKey } from "interfaces/errors";

import { generateSecretErrMsg } from "pages/SoftwarePage/helpers";

import {
  ADD_SOFTWARE_ERROR_PREFIX,
  DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE,
  REQUEST_TIMEOUT_ERROR_MESSAGE,
  ensurePeriod,
  formatAlreadyAvailableInstallMessage,
} from "../../helpers";

export const getFleetAppPolicyName = (appName: string) => {
  return `[Install software] ${appName}`;
};

export const getFleetAppPolicyDescription = (appName: string) => {
  return `Policy triggers automatic install of ${appName} on each host that's missing this software.`;
};

export const getErrorMessage = (err: unknown) => {
  const responseStatus =
    (isAxiosError(err) ? err.response?.status : undefined) ??
    (hasStatusKey(err) ? err.status : undefined);
  const isTimeout =
    responseStatus === 504 || responseStatus === 408 || responseStatus === 499; // upstream proxy/LB canceled the request
  const reason = getErrorReason(err);

  if (
    isTimeout ||
    reason.includes("json decoder error") // 400 bad request when really slow
  ) {
    return REQUEST_TIMEOUT_ERROR_MESSAGE;
  }

  // Server returns a complete user-facing message; pass it through as-is.
  if (reason.includes("can be added to the same fleet")) {
    return ensurePeriod(reason);
  }

  // software is already available for install
  if (reason.toLowerCase().includes("already")) {
    const alreadyAvailableMessage = formatAlreadyAvailableInstallMessage(
      reason
    );
    if (alreadyAvailableMessage) {
      return alreadyAvailableMessage;
    }
  }

  if (reason.includes("Secret variable")) {
    return generateSecretErrMsg(err);
  }
  if (reason) {
    return `${ADD_SOFTWARE_ERROR_PREFIX} ${ensurePeriod(reason)}`;
  }

  return DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE;
};

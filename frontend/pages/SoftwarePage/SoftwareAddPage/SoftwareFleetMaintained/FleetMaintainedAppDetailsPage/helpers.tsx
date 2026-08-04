import { isAxiosError } from "axios";
import { getErrorReason } from "interfaces/errors";

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
  const isTimeout =
    isAxiosError(err) &&
    (err.response?.status === 504 || err.response?.status === 408);
  const reason = getErrorReason(err);

  if (
    isTimeout ||
    reason.includes("json decoder error") // 400 bad request when really slow
  ) {
    return REQUEST_TIMEOUT_ERROR_MESSAGE;
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

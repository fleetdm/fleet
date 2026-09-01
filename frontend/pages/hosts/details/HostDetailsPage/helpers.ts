import { getErrorReason } from "interfaces/errors";
import { IHost } from "interfaces/host";
import { isAndroid, isChrome } from "interfaces/platform";

import { getHostDeviceStatusUIState } from "../helpers";

const DEFAULT_ERROR_MESSAGE = "refetch error.";

export const getErrorMessage = (e: unknown, hostName: string) => {
  let errorMessage = getErrorReason(e, {
    reasonIncludes: "Host does not have MDM turned on",
  });

  if (!errorMessage) {
    errorMessage = DEFAULT_ERROR_MESSAGE;
  }

  return `Host "${hostName}" ${errorMessage}`;
};

// The "My device" link opens the end-user page authed by the host's device
// auth token. Fleet Desktop is what mints that token on orbit check-in, so a
// host missing fleet_desktop_version is also missing a token and has no live
// end-user surface. Hide the button on wiped hosts and on hosts with a wipe
// in flight — the device is about to have no end-user session to review.
export const canShowMyDeviceButton = (
  host: Pick<IHost, "fleet_desktop_version" | "mdm" | "platform">
) => {
  // Android and ChromeOS have no My device page, so the link would only lead to
  // an error. GET /hosts/:id/device_url rejects them for the same reason.
  if (isAndroid(host.platform) || isChrome(host.platform)) return false;
  if (!host.fleet_desktop_version) return false;
  const uiState = getHostDeviceStatusUIState(
    host.mdm.device_status,
    host.mdm.pending_action
  );
  return uiState !== "wiped" && uiState !== "wiping";
};

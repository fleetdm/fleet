import { getErrorReason } from "interfaces/errors";
import { IHost } from "interfaces/host";
import { isAndroid, isChrome, isIPadOrIPhone } from "interfaces/platform";
import { INITIAL_FLEET_DATE } from "utilities/constants";

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

// The "My device" link opens the end-user page. macOS/Windows/Linux reach it
// with a device auth token minted by Fleet Desktop on orbit check-in, so a host
// missing fleet_desktop_version has no live end-user surface. iOS/iPadOS don't
// run Fleet Desktop and reach it by host UUID instead. Hide the button on wiped
// hosts and on hosts with a wipe in flight — the device is about to have no
// end-user session to review.
export const canShowMyDeviceButton = (
  // platform is a plain string rather than HostPlatform: legacy ChromeOS hosts
  // report "CrOS", which predates the HostPlatform union.
  host: Pick<IHost, "fleet_desktop_version" | "mdm"> & { platform: string }
) => {
  // Android and ChromeOS have no My device page, so the link would only lead to
  // an error. GET /hosts/:id/device_url rejects them for the same reason.
  if (
    isAndroid(host.platform) ||
    isChrome(host.platform) ||
    host.platform === "CrOS"
  ) {
    return false;
  }
  if (!isIPadOrIPhone(host.platform) && !host.fleet_desktop_version) {
    return false;
  }
  const uiState = getHostDeviceStatusUIState(
    host.mdm.device_status,
    host.mdm.pending_action
  );
  return uiState !== "wiped" && uiState !== "wiping";
};

// Hosts created in a pending state (Apple Business Manager, Windows Autopilot) are inserted with
// refetch_requested already set, but there is nothing on the device yet to answer it, so the flag
// stays on until the host actually enrolls. Their last_enrolled_at holds the "never" sentinel until
// then, which is what separates them from hosts that can return vitals.
export const hasEverEnrolled = (host: Pick<IHost, "last_enrolled_at">) =>
  !!host.last_enrolled_at && host.last_enrolled_at >= INITIAL_FLEET_DATE;

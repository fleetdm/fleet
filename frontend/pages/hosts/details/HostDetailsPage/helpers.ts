import { getErrorReason } from "interfaces/errors";
import { IHost } from "interfaces/host";

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
// end-user surface. A wiped host has no end-user session to review either.
export const canShowMyDeviceButton = (
  host: Pick<IHost, "fleet_desktop_version" | "mdm">
) => !!host.fleet_desktop_version && host.mdm.device_status !== "wiped";

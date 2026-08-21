import { ISetupStep } from "interfaces/setup";
import { SCRIPT_PACKAGE_SOURCES } from "interfaces/software";

const DEFAULT_ERROR_MESSAGE = "refetch error.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown, hostName: string) => {
  return `Host "${hostName}" ${DEFAULT_ERROR_MESSAGE}`;
};

export const hasRemainingSetupSteps = (
  statuses: ISetupStep[] | null | undefined
) => {
  if (!statuses || statuses.length === 0) {
    // not configured or no software selected
    return false;
  }

  return statuses.some((s) => ["pending", "running"].includes(s.status));
};

export const getFailedSoftwareInstall = (
  statuses: ISetupStep[] | null | undefined
): ISetupStep | null => {
  if (!statuses || statuses.length === 0) {
    // not configured or no software selected
    return null;
  }

  const failedSoftware = statuses.filter(
    (s) =>
      (s.type === "software_install" || s.type === "software_script_run") &&
      s.status === "failure"
  );
  if (failedSoftware.length === 0) {
    return null;
  }
  // Find the first one with an error message, otherwise return the first one.
  const firstWithError = failedSoftware.find((s) => s.error);
  return firstWithError ?? failedSoftware[0];
};

/** Checks if the software is a script-only package (sh, ps1, or py)
 * by examining the source field from the API */
export const isSoftwareScriptSetup = (s: ISetupStep) => {
  if (!s.source) return false;

  return SCRIPT_PACKAGE_SOURCES.includes(s.source);
};

// Hosts after enrollment during which we suppress the "host is offline" banner.
// Orbit endpoints do not update host_seen_times, so a freshly enrolled host can appear offline
// until its first osquery distributed-read check-in (typically within 5-10 minutes).
const RECENTLY_ENROLLED_THRESHOLD_MS = 10 * 60 * 1000;

export const isRecentlyEnrolled = (
  lastEnrolledAt: string | undefined
): boolean => {
  if (!lastEnrolledAt) return false;
  const enrolledAt = new Date(lastEnrolledAt).getTime();
  if (isNaN(enrolledAt)) return false;
  // Require a non-negative delta so a future timestamp (e.g. from client/server clock skew) is not
  // treated as "recent" and does not hide a real offline state indefinitely.
  const delta = Date.now() - enrolledAt;
  return delta >= 0 && delta < RECENTLY_ENROLLED_THRESHOLD_MS;
};

// Same solution as defined in /templates/enroll-ota.html (https://github.com/fleetdm/fleet/pull/26592)
export const isIPhone = (navigator: Navigator) =>
  /iPhone/i.test(navigator.userAgent);
export const isIPad = (navigator: Navigator) =>
  /iPad/i.test(navigator.userAgent) ||
  (/Macintosh/i.test(navigator.userAgent) &&
    navigator.maxTouchPoints !== undefined &&
    navigator.maxTouchPoints > 1);
// Android does not have access to this UI
export const isMac = (navigator: Navigator) =>
  (/Macintosh/i.test(navigator.userAgent) && !isIPad) ||
  /Mac OS X/i.test(navigator.userAgent);

/** The shape `sendRequest` rejects with: the Axios *response*, so the API's
 * error body sits on `data`. */
interface IDeviceAPIError {
  status?: number;
  data?: { sso_required?: boolean };
}

/** With Fleet Desktop SSO on, the device API answers 401 both for a stale
 * device token and for a request carrying no IdP session. Only the second sets
 * `sso_required`, and only it should send the end user to the IdP: the initiate
 * endpoint authenticates with the device token too, so a round-trip cannot fix
 * the first. */
export const isSSORequiredError = (error: unknown): boolean => {
  const response = error as IDeviceAPIError | null | undefined;
  return response?.status === 401 && response?.data?.sso_required === true;
};

const ssoAttemptKey = (deviceAuthToken: string) =>
  `fleet-device-sso-attempt:${deviceAuthToken}`;

/** One automatic trip to the IdP per token. Coming back still unauthenticated
 * means the session cookie never stuck (blocked cookies, clock skew), and
 * initiating again would bounce the end user between Fleet and the IdP forever.
 *
 * Reads and writes are guarded because browsers configured to block site data
 * throw on access; losing the flag costs the loop guard, which is not worth
 * taking the page down for. */
export const hasAttemptedDeviceSSO = (deviceAuthToken: string): boolean => {
  try {
    return sessionStorage.getItem(ssoAttemptKey(deviceAuthToken)) !== null;
  } catch {
    return false;
  }
};

export const recordDeviceSSOAttempt = (deviceAuthToken: string) => {
  try {
    sessionStorage.setItem(ssoAttemptKey(deviceAuthToken), "1");
  } catch {
    // see hasAttemptedDeviceSSO
  }
};

export const clearDeviceSSOAttempt = (deviceAuthToken: string) => {
  try {
    sessionStorage.removeItem(ssoAttemptKey(deviceAuthToken));
  } catch {
    // see hasAttemptedDeviceSSO
  }
};

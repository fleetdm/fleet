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

interface IDeviceAPIError {
  status?: number;
  data?: { sso_required?: boolean };
}

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
 * Unreadable storage also answers false, not just a spent attempt: the guard
 * only works if the flag survives a full page navigation, so a browser that
 * won't store it cannot be given automatic attempts at all. Access is guarded
 * because browsers configured to block site data throw on it. That
 * configuration usually blocks the SSO session cookie too, which is exactly
 * when the loop would run. Signing in by hand still works; it just costs a
 * click. */
export const canAutoInitiateDeviceSSO = (deviceAuthToken: string): boolean => {
  try {
    return sessionStorage.getItem(ssoAttemptKey(deviceAuthToken)) === null;
  } catch {
    return false;
  }
};

/** Returns whether the attempt will still be there after the trip to the IdP. */
export const recordDeviceSSOAttempt = (deviceAuthToken: string): boolean => {
  const key = ssoAttemptKey(deviceAuthToken);
  try {
    sessionStorage.setItem(key, "1");
    return sessionStorage.getItem(key) !== null;
  } catch {
    return false;
  }
};

/** Returns whether an automatic attempt is available again, so a browser that
 * cannot store the flag is not handed a fresh automatic attempt by a
 * successful load. */
export const clearDeviceSSOAttempt = (deviceAuthToken: string): boolean => {
  try {
    sessionStorage.removeItem(ssoAttemptKey(deviceAuthToken));
    return true;
  } catch {
    return false;
  }
};

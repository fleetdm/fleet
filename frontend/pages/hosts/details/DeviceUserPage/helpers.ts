import { ISetupStep } from "interfaces/setup";

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

/** Checks if the software is a script-only package (sh or ps1)
 * by examining the source field from the API */
export const isSoftwareScriptSetup = (s: ISetupStep) => {
  if (!s.source) return false;

  return s.source === "sh_packages" || s.source === "ps1_packages";
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

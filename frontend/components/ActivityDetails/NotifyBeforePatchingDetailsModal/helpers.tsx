import React from "react";

import { pluralize } from "utilities/strings/stringUtils";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

// TODO: swap in the "another notification displayed" exit code once published.
// The no-script_execution_id branch already handles the dispatcher deferral.
export const DEFERRED_EXIT_CODE_TBC = -1;

export const FAILURE_COPY_BY_EXIT_CODE: Record<number, string> = {
  0: "If the host is offline when the patch is forced, Fleet skips the patch. When the host comes back online Fleet notifies the end user again and the patch is forced 1 hour later.",
  30: "The notification couldn't load. Fleet will try again on the next policy run.",
  31: "The notification couldn't load. Fleet will try again on the next policy run.",
  41: "The screen was locked so the end user couldn't see the notification. Fleet will try again on the next policy run.",
  100: "The Fleet Desktop app is required to notify end users. Add the app from the Fleet-maintained catalog and deploy to all your hosts.",
  101: "The Fleet Desktop app v1.5.0 is required to notify end users. Add the app from the Fleet-maintained catalog and deploy to all your hosts.",
  [DEFERRED_EXIT_CODE_TBC]:
    "Another notification was displayed. Fleet will try again on the next policy run.",
};

export const DEFERRED_SENTENCE =
  "Another notification was displayed. Fleet will try again on the next policy run.";

// Covers both success (exit 0 caveat) and failure sentences.
export const getCaveatSentence = (
  scriptExecutionId?: string,
  exitCode?: number | null
): string | null => {
  // Absence of script_execution_id signals a server-side deferral.
  if (!scriptExecutionId) return DEFERRED_SENTENCE;
  if (exitCode === null || exitCode === undefined) return null;
  return FAILURE_COPY_BY_EXIT_CODE[exitCode] ?? null;
};

// Longer-form success caveat; intentionally different from
// FAILURE_COPY_BY_EXIT_CODE[0].
export const getAutomationNotifiedSentence = (timeBefore?: number): string => {
  const label = timeBefore === 300 ? "5 minutes" : "1 hour";
  return `End user was notified. Patch will be forced in ${label}. If the host is offline when a patch should be forced, Fleet notifies the end user again when it comes back online and patches it after 1 hour.`;
};

export const SKIPPED_INSTALL_NOTIFY_EXPLANATION =
  "The app was open. The end user will be notified before the patch is forced.";

// Shared with the "End user experience" dropdown — keep in sync.
export const PATCHING_END_USER_EXPERIENCE_URL =
  "https://fleetdm.com/learn-more-about/patching-end-user-experience";

// Exit codes that append the "End user experience" learn-more link.
export const EXIT_CODES_NEEDING_EUE_LINK: ReadonlySet<number> = new Set([
  100,
  101,
]);

// Substring the BE appends only on notify_before_patching skips.
const NOTIFY_SKIP_MARKER = "Fleet notifies the end user";
export const isNotifyBeforePatchingSkip = (
  preInstallOutput?: string | null
): boolean => !!preInstallOutput?.includes(NOTIFY_SKIP_MARKER);

// Bold titles, Oxford comma, ", and N more app(s)" past three.
export const renderNotifyTitleList = (titles: string[]): React.ReactNode => {
  const bold = (name: string) => (
    <strong>{getDisplayedSoftwareName(name)}</strong>
  );
  const overflow = titles.length - 3;
  if (titles.length === 1) return bold(titles[0]);
  if (titles.length === 2) {
    return (
      <>
        {bold(titles[0])} and {bold(titles[1])}
      </>
    );
  }
  if (titles.length === 3) {
    return (
      <>
        {bold(titles[0])}, {bold(titles[1])}, and {bold(titles[2])}
      </>
    );
  }
  if (titles.length > 3) {
    return (
      <>
        {bold(titles[0])}, {bold(titles[1])}, {bold(titles[2])}, and {overflow}{" "}
        more {pluralize(overflow, "app")}
      </>
    );
  }
  return null;
};

import React from "react";

import { INotifyActivityStatus } from "interfaces/activity";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

export const isNotifyFailure = (
  status: INotifyActivityStatus | undefined
): boolean => status === "failed";

export const DEFERRED_EXIT_CODE = 50;

// Titles inlined into the intro sentence; past this we truncate to
// "..., and N more apps" and move the full list to the Apps row.
export const INLINE_APP_LIMIT = 4;

// Keyed by script exit code. Index 0 is the offline caveat shown on success;
// non-zero entries are failure reasons.
export const COPY_BY_EXIT_CODE: Record<number, string> = {
  0: "If the host is offline when the patch is forced, Fleet skips the patch. When the host comes back online Fleet notifies the end user again and the patch is forced 1 hour later.",
  30: "The notification couldn't load. Fleet will try again on the next policy run.",
  31: "The notification couldn't load. Fleet will try again on the next policy run.",
  41: "The screen was locked so the end user couldn't see the notification. Fleet will try again on the next policy run.",
  [DEFERRED_EXIT_CODE]:
    "Another notification was displayed. Fleet will try again on the next policy run.",
  100: "The Fleet Desktop app is required to notify end users. Add the app from the Fleet-maintained catalog and deploy to all your hosts.",
  101: "The Fleet Desktop app v1.5.0 is required to notify end users. Add the app from the Fleet-maintained catalog and deploy to all your hosts.",
};

export const DEFERRED_SENTENCE =
  "Another notification was displayed. Fleet will try again on the next policy run.";

// Covers both success (exit 0 caveat) and failure sentences.
export const getCaveatMessage = (
  failed: boolean,
  scriptExecutionId?: string,
  exitCode?: number | null
): string | null => {
  // Absence of script_execution_id signals a server-side deferral, which the
  // BE only emits as a failure. Guard against showing a caveat next to a
  // success icon if the invariant ever slips.
  if (!scriptExecutionId) return failed ? DEFERRED_SENTENCE : null;
  if (exitCode === null || exitCode === undefined) return null;
  // Mirror guard: exit 0 is the success caveat; don't show it next to a red icon.
  if (failed && exitCode === 0) return null;
  return COPY_BY_EXIT_CODE[exitCode] ?? null;
};

// One source of truth for the "1 hour" / "5 minutes" mapping.
export const formatNotifyTimeLabel = (timeBefore?: number): string =>
  timeBefore === 300 ? "5 minutes" : "1 hour";

// Longer-form success caveat; intentionally different from COPY_BY_EXIT_CODE[0].
export const getAutomationNotifiedMessage = (timeBefore?: number): string =>
  `End user was notified. Patch will be forced in ${formatNotifyTimeLabel(
    timeBefore
  )}. If the host is offline when a patch should be forced, Fleet notifies the end user again when it comes back online and patches it after 1 hour.`;

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

// Bold titles, Oxford comma. Inline all when ≤ INLINE_APP_LIMIT; past that
// truncate to 3 + ", and N more apps". Returns null on empty input.
// TODO(BE contract): software_titles is currently string[]; if BE emits
// objects with display_name, swap this to render the display name.
export const renderNotifyTitleList = (titles: string[]): React.ReactNode => {
  if (titles.length === 0) return null;
  const bold = (name: string) => (
    <strong>{getDisplayedSoftwareName(name)}</strong>
  );
  if (titles.length === 1) return bold(titles[0]);
  if (titles.length === 2) {
    return (
      <>
        {bold(titles[0])} and {bold(titles[1])}
      </>
    );
  }
  if (titles.length <= INLINE_APP_LIMIT) {
    const head = titles.slice(0, -1);
    const tail = titles[titles.length - 1];
    return (
      <>
        {head.map((t, i) => (
          <React.Fragment key={t}>
            {bold(t)}
            {i < head.length - 1 ? ", " : ", and "}
          </React.Fragment>
        ))}
        {bold(tail)}
      </>
    );
  }
  // overflow is always ≥ 2 past INLINE_APP_LIMIT, so no pluralize needed.
  const overflow = titles.length - 3;
  return (
    <>
      {bold(titles[0])}, {bold(titles[1])}, {bold(titles[2])}, and {overflow}{" "}
      more apps
    </>
  );
};

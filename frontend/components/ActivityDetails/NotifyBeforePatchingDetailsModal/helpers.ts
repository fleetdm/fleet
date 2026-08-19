// TODO(#50915): Marko will publish a "another notification is displayed" exit
// code — waiting on the number. Meanwhile the "no script_execution_id" branch
// covers the dispatcher-caught case (see spec: absence of execution_id is
// unambiguous because the patch kind sets no `expires_at`).
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

// The exit-code map covers both success (exit 0 shows the offline caveat) and
// failure sentences — the intro's verb changes, but the exit-code-driven
// sentence renders in both cases.
export const getCaveatSentence = (
  scriptExecutionId?: string,
  exitCode?: number | null
): string | null => {
  // A server-side deferral emits an activity with no script run — absence of
  // script_execution_id is the unambiguous signal.
  if (!scriptExecutionId) return DEFERRED_SENTENCE;
  if (exitCode === null || exitCode === undefined) return null;
  return FAILURE_COPY_BY_EXIT_CODE[exitCode] ?? null;
};

// Longer-form caveat used by the policy automation modal's success case. The
// activity feed modal uses FAILURE_COPY_BY_EXIT_CODE[0] instead — Figma treats
// them as different sentences by design.
export const getAutomationNotifiedSentence = (timeBefore?: number): string => {
  const label = timeBefore === 300 ? "5 minutes" : "1 hour";
  return `End user was notified. Patch will be forced in ${label}. If the host is offline when a patch should be forced, Fleet notifies the end user again when it comes back online and patches it after 1 hour.`;
};

export const SKIPPED_INSTALL_NOTIFY_EXPLANATION =
  "The app was open. The end user will be notified before the patch is forced.";

// Same URL the "End user experience" dropdown in DeployModal / PolicyForm
// links to — kept in sync so admins land on the same doc regardless of where
// they discover the "Fleet Desktop required" sentence.
export const PATCHING_END_USER_EXPERIENCE_URL =
  "https://fleetdm.com/learn-more-about/patching-end-user-experience";

// Exit codes whose sentence needs the "End user experience" learn-more link
// appended — currently the two Fleet-Desktop-required variants (missing app,
// too-old version). Kept as a set so future exit codes can opt in explicitly.
export const EXIT_CODES_NEEDING_EUE_LINK: ReadonlySet<number> = new Set([
  100,
  101,
]);

// The BE emits the pre-install query output with a notify-specific sentence
// appended for the notify_before_patching variant. That substring is the
// unambiguous signal we're looking at a notify skip vs a patch_when_closed
// skip. Kept as a constant so the check doesn't drift silently.
const NOTIFY_SKIP_MARKER = "Fleet notifies the end user";
export const isNotifyBeforePatchingSkip = (
  preInstallOutput?: string | null
): boolean => !!preInstallOutput?.includes(NOTIFY_SKIP_MARKER);

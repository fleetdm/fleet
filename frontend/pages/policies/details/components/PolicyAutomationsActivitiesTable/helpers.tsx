import { ActivityType } from "interfaces/activity";
import { IPolicyAutomationActivity } from "interfaces/policy";
import { Colors } from "styles/var/colors";

const withName = (base: string, name?: string) =>
  name ? `${base} (${name})` : base;

const getNotifySoftwareName = (
  details: IPolicyAutomationActivity["details"] | undefined
): string | undefined => {
  // BE joins per-policy so each row usually carries a singular software_title;
  // fall back to the first entry of the plural payload field just in case.
  return details?.software_title || details?.software_titles?.[0];
};

const getNotifiedSentence = (timeBefore?: number): string => {
  const label = timeBefore === 300 ? "5 minutes" : "1 hour";
  return `End user was notified. Patch will be forced in ${label}. If the host is offline when a patch should be forced, Fleet notifies the end user again when it comes back online and patches it after 1 hour.`;
};

/**
 * Human-readable label for the "Automation" column. Failure rows mirror the
 * success wording with "failed" — e.g. "Software installed (1Password)" vs
 * "Software failed (1Password)".
 */
export const getAutomationRunDisplayName = (
  activity: IPolicyAutomationActivity
): string => {
  const { type, status, details } = activity;
  const failed = status === "error";

  switch (type) {
    case ActivityType.InstalledSoftware:
    case ActivityType.InstalledAppStoreApp:
      // A patch-when-closed skip is recorded as a failed_install, but it was
      // deferred because the app was open — not a failure. Label it distinctly,
      // matching the activity feed and install-details treatment.
      if (details?.skipped_install) {
        return withName("Patch skipped", details?.software_title);
      }
      return withName(
        failed ? "Software failed" : "Software installed",
        details?.software_title
      );
    case ActivityType.NotifiedEndUserBeforePatching:
      return withName(
        failed ? "Failed to notify" : "End user notified",
        getNotifySoftwareName(details)
      );
    case ActivityType.RanScript:
      return withName(
        failed ? "Script failed" : "Script ran",
        details?.script_name
      );
    case ActivityType.RanAutomationCalendarEvent:
      return "Calendar event created";
    case ActivityType.FailedAutomationCalendarEvent:
      return "Calendar event failed";
    case ActivityType.RanAutomationConditionalAccess:
      return "Single sign-on blocked";
    case ActivityType.FailedAutomationConditionalAccess:
      return "Single sign-on failed";
    case ActivityType.RanAutomationWebhook:
      return "Webhook queued";
    case ActivityType.FailedAutomationWebhook:
      return "Webhook failed";
    case ActivityType.RanAutomationTicket:
      return "Ticket queued";
    case ActivityType.FailedAutomationTicket:
      return "Ticket failed";
    default:
      return failed ? "Automation failed" : "Automation ran";
  }
};

/** Status icon paired with an automation outcome: a patch-when-closed skip and
 *  a successful "end user notified" both use the muted grey "!" glyph — the
 *  policy didn't succeed at patching, but Fleet handled it deliberately.
 *  Red outline for other failures, green for successes. */
export const getAutomationStatusIcon = (
  activity: IPolicyAutomationActivity
): { name: "error-outline" | "success-outline"; color?: Colors } => {
  if (activity.details?.skipped_install) {
    return { name: "error-outline", color: "ui-fleet-black-50" };
  }
  if (
    activity.type === ActivityType.NotifiedEndUserBeforePatching &&
    activity.status !== "error"
  ) {
    return { name: "error-outline", color: "ui-fleet-black-50" };
  }
  return activity.status === "error"
    ? { name: "error-outline" }
    : { name: "success-outline" };
};

/**
 * Text shown in the "Details" column (and the modal's primary block): the
 * remote error response for failures, or the script/install output for the
 * task activities. Empty when neither applies.
 */
export const getDetailOutputText = (
  activity: IPolicyAutomationActivity
): string => {
  // Success (or deferral) rows for the notify activity render a computed
  // sentence keyed on time_before — the reminder swap 1hr→5min lands here.
  // Failures fall through to activity.output so the notification script's raw
  // output shows in the row preview; the modal fetches the script result and
  // maps the exit code to a human sentence.
  if (
    activity.type === ActivityType.NotifiedEndUserBeforePatching &&
    activity.status !== "error"
  ) {
    return getNotifiedSentence(activity.details?.time_before);
  }
  if (activity.status === "error" && activity.details?.error_response) {
    return activity.details.error_response;
  }
  // For software installs, the install-script output is the primary preview, but
  // a failure at the pre-install query or post-install script stage leaves it
  // empty — fall back to those so the row still shows the failing stage's output.
  // Other activity types have null pre/post output, so this is just `output`.
  return (
    activity.output ||
    activity.post_install_output ||
    activity.pre_install_output ||
    ""
  );
};

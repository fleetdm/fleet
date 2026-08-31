import { ActivityType } from "interfaces/activity";
import { IPolicyAutomationActivity } from "interfaces/policy";
import { SKIPPED_INSTALL_DETAILS } from "components/ActivityDetails/InstallDetails/constants";
import { Colors } from "styles/var/colors";

const withName = (base: string, name?: string) =>
  name ? `${base} (${name})` : base;

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
    case ActivityType.ResentConfigurationProfile:
      // A resend is only recorded once the profile is queued for redelivery, so this row is
      // always a success; whether the profile then verifies shows on the host, not here.
      return withName("Configuration profile resent", details?.profile_name);
    default:
      return failed ? "Automation failed" : "Automation ran";
  }
};

/** Status icon paired with an automation outcome: a patch-when-closed skip is
 *  the same "!" glyph as a failure but muted grey (deferred, not a failure),
 *  a red outline for other failures, and a green one for successes. */
export const getAutomationStatusIcon = (
  activity: IPolicyAutomationActivity
): { name: "error-outline" | "success-outline"; color?: Colors } => {
  if (activity.details?.skipped_install) {
    return { name: "error-outline", color: "ui-fleet-black-50" };
  }
  return activity.status === "error"
    ? { name: "error-outline" }
    : { name: "success-outline" };
};

/**
 * Text shown in the "Details" column: the explanation for a deferred patch, the
 * remote error response for failures, or the script/install output for the task
 * activities. Empty when none apply.
 */
export const getDetailOutputText = (
  activity: IPolicyAutomationActivity
): string => {
  if (activity.details?.skipped_install) {
    return SKIPPED_INSTALL_DETAILS;
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

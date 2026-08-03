import { ActivityType } from "interfaces/activity";
import {
  IPolicyAutomationActivity,
  PolicyAutomationActivityStatus,
} from "interfaces/policy";

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
    default:
      return failed ? "Automation failed" : "Automation ran";
  }
};

/** Status icon paired with an automation outcome: a red outline for failures,
 *  a green one for successes. */
export const getAutomationStatusIconName = (
  status: PolicyAutomationActivityStatus
): "error-outline" | "success-outline" =>
  status === "error" ? "error-outline" : "success-outline";

/**
 * Text shown in the "Details" column (and the modal's primary block): the
 * remote error response for failures, or the script/install output for the
 * task activities. Empty when neither applies.
 */
export const getDetailOutputText = (
  activity: IPolicyAutomationActivity
): string => {
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

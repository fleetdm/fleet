import { ActivityType } from "interfaces/activity";
import { IPolicyAutomationActivity } from "interfaces/policy";
import { Colors } from "styles/var/colors";

import { getAutomationNotifiedMessage } from "components/ActivityDetails/NotifyBeforePatchingDetailsModal/helpers";

const withName = (base: string, name?: string) =>
  name ? `${base} (${name})` : base;

// BE joins per-policy so rows usually carry a singular software_title;
// software_titles[0] is a safety net.
const getNotifySoftwareName = (
  details: IPolicyAutomationActivity["details"] | undefined
): string | undefined =>
  details?.software_title || details?.software_titles?.[0];

/** Label for the "Automation" column. */
export const getAutomationRunDisplayName = (
  activity: IPolicyAutomationActivity
): string => {
  const { type, status, details } = activity;
  const failed = status === "error";

  switch (type) {
    case ActivityType.InstalledSoftware:
    case ActivityType.InstalledAppStoreApp:
      // App-open skips are recorded as failed_install but aren't failures.
      if (details?.skipped_install) {
        return withName("Patch skipped", details?.software_title);
      }
      return withName(
        failed ? "Software failed" : "Software installed",
        details?.software_title
      );
    case ActivityType.NotifiedEndUserBeforePatching:
      return withName(
        failed ? "Failed to notify" : "Notified end user",
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

/** Grey "!" for deliberate non-successes (app-open skips, notify success);
 *  red for failures, green for successes. */
export const getAutomationStatusIcon = (
  activity: IPolicyAutomationActivity
): { name: "error-outline" | "success-outline"; color?: Colors } => {
  if (activity.details?.skipped_install) {
    return { name: "error-outline", color: "ui-fleet-black-50" };
  }
  if (
    activity.type === ActivityType.NotifiedEndUserBeforePatching &&
    activity.status === "success"
  ) {
    return { name: "error-outline", color: "ui-fleet-black-50" };
  }
  return activity.status === "error"
    ? { name: "error-outline" }
    : { name: "success-outline" };
};

/** Text for the "Details" column preview. */
export const getDetailOutputText = (
  activity: IPolicyAutomationActivity
): string => {
  // Notify success/deferral: computed sentence keyed on time_before.
  if (
    activity.type === ActivityType.NotifiedEndUserBeforePatching &&
    activity.status === "success"
  ) {
    return getAutomationNotifiedMessage(activity.details?.time_before);
  }
  if (activity.status === "error" && activity.details?.error_response) {
    return activity.details.error_response;
  }
  // Fall back through the install stages so the failing stage's output shows.
  return (
    activity.output ||
    activity.post_install_output ||
    activity.pre_install_output ||
    ""
  );
};

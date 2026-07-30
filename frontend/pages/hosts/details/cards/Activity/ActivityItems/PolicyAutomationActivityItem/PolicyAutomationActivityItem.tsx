import React from "react";

import { ActivityType, IHostPastActivityType } from "interfaces/activity";
import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "policy-automation-activity-item";

type PolicyAutomationActivityType =
  | ActivityType.RanAutomationWebhook
  | ActivityType.RanAutomationTicket
  | ActivityType.RanAutomationCalendarEvent
  | ActivityType.RanAutomationConditionalAccess
  | ActivityType.FailedAutomationWebhook
  | ActivityType.FailedAutomationTicket
  | ActivityType.FailedAutomationCalendarEvent
  | ActivityType.FailedAutomationConditionalAccess;

const AUTOMATION_COPY: Record<PolicyAutomationActivityType, string> = {
  [ActivityType.RanAutomationWebhook]:
    "sent a webhook because this host failed a policy.",
  [ActivityType.RanAutomationTicket]:
    "created a ticket because this host failed a policy.",
  [ActivityType.RanAutomationCalendarEvent]:
    "created a calendar event because this host failed a policy.",
  [ActivityType.RanAutomationConditionalAccess]:
    "blocked single sign-on because this host failed a policy.",
  [ActivityType.FailedAutomationWebhook]:
    "failed to send a webhook after this host failed a policy.",
  [ActivityType.FailedAutomationTicket]:
    "failed to create a ticket after this host failed a policy.",
  [ActivityType.FailedAutomationCalendarEvent]:
    "failed to create a calendar event after this host failed a policy.",
  [ActivityType.FailedAutomationConditionalAccess]:
    "failed to block single sign-on after this host failed a policy.",
};

const isPolicyAutomationActivityType = (
  type: IHostPastActivityType
): type is PolicyAutomationActivityType => type in AUTOMATION_COPY;

const PolicyAutomationActivityItem = ({
  activity,
  isSoloActivity,
}: IHostActivityItemComponentProps) => {
  if (!isPolicyAutomationActivityType(activity.type)) {
    return null;
  }

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      isSoloActivity={isSoloActivity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.fleet_initiated ? "Fleet" : activity.actor_full_name}</b>{" "}
      {AUTOMATION_COPY[activity.type]}
    </ActivityItem>
  );
};

export default PolicyAutomationActivityItem;

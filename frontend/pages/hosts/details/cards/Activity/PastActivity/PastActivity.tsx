import React from "react";
import ReactTooltip from "react-tooltip";
import { formatDistanceToNowStrict } from "date-fns";

import Avatar from "components/Avatar";
import Icon from "components/Icon";
import Button from "components/buttons/Button";

import { COLORS } from "styles/var/colors";
import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";
import {
  addGravatarUrlToResource,
  formatScriptNameForActivityItem,
  internationalTimeFormat,
} from "utilities/helpers";
import { IActivity } from "interfaces/activity";
import { ShowActivityDetailsHandler } from "../Activity";

const baseClass = "past-activity";

interface IPastActivityProps {
  activity: IActivity;
  // TODO: To handle clicks for different activity types, this could be refactored as a reducer that
  // takes the activity and dispatches the relevant show details action based on the activity type
  onDetailsClick: ShowActivityDetailsHandler;
}

const RanScriptActivityDetails = ({
  activity,
  onDetailsClick,
}: Pick<IPastActivityProps, "activity" | "onDetailsClick">) => (
  <span className={`${baseClass}__details-topline`}>
    <b>{activity.actor_full_name}</b>
    <>
      {" "}
      ran {formatScriptNameForActivityItem(activity.details?.script_name)} on
      this host.{" "}
      <Button
        className={`${baseClass}__show-query-link`}
        variant="text-link"
        onClick={() => onDetailsClick?.(activity)}
      >
        Show details{" "}
        <Icon className={`${baseClass}__show-query-icon`} name="eye" />
      </Button>
    </>
  </span>
);

const LockedHostActivityDetails = ({
  activity,
}: Pick<IPastActivityProps, "activity">) => (
  <span className={`${baseClass}__details-topline`}>
    <b>{activity.actor_full_name}</b> locked this host.
  </span>
);

const UnlockedHostActivityDetails = ({
  activity,
}: Pick<IPastActivityProps, "activity">) => (
  <span className={`${baseClass}__details-topline`}>
    <b>{activity.actor_full_name}</b>{" "}
    {activity.details?.host_platform === "darwin"
      ? "viewed the six-digit unlock PIN for"
      : "unlocked"}{" "}
    this host.
  </span>
);

const PastActivityTopline = ({
  activity,
  onDetailsClick,
}: IPastActivityProps) => {
  switch (activity.type) {
    case "ran_script":
      return (
        <RanScriptActivityDetails
          activity={activity}
          onDetailsClick={onDetailsClick}
        />
      );
    case "locked_host":
      return <LockedHostActivityDetails activity={activity} />;
    case "unlocked_host":
      return <UnlockedHostActivityDetails activity={activity} />;
    default:
      return null;
  }
};

// TODO: Combine this with ./UpcomingActivity/UpcomingActivity.tsx and
// frontend/pages/DashboardPage/cards/ActivityFeed/ActivityItem/ActivityItem.tsx
const PastActivity = ({ activity, onDetailsClick }: IPastActivityProps) => {
  const { actor_email } = activity;
  const { gravatar_url } = actor_email
    ? addGravatarUrlToResource({ email: actor_email })
    : { gravatar_url: DEFAULT_GRAVATAR_LINK };
  const activityCreatedAt = new Date(activity.created_at);

  return (
    <div className={baseClass}>
      <Avatar
        className={`${baseClass}__avatar-image`}
        user={{ gravatar_url }}
        size="small"
        hasWhiteBackground
      />
      <div className={`${baseClass}__details-wrapper`}>
        <div className="activity-details">
          <PastActivityTopline
            activity={activity}
            onDetailsClick={onDetailsClick}
          />
          <br />
          <span
            className={`${baseClass}__details-bottomline`}
            data-tip
            data-for={`activity-${activity.id}`}
          >
            {formatDistanceToNowStrict(activityCreatedAt, {
              addSuffix: true,
            })}
          </span>
          <ReactTooltip
            className="date-tooltip"
            place="top"
            type="dark"
            effect="solid"
            id={`activity-${activity.id}`}
            backgroundColor={COLORS["tooltip-bg"]}
          >
            {internationalTimeFormat(activityCreatedAt)}
          </ReactTooltip>
        </div>
      </div>
      <div className={`${baseClass}__dash`} />
    </div>
  );
};

export default PastActivity;

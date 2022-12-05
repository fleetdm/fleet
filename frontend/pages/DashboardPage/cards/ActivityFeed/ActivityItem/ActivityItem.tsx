import React from "react";
import { find, lowerCase, noop } from "lodash";
import { formatDistanceToNowStrict } from "date-fns";

import { ActivityType, IActivity, IActivityDetails } from "interfaces/activity";
import { addGravatarUrlToResource } from "utilities/helpers";
import Avatar from "components/Avatar";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "activity-item";

const DEFAULT_GRAVATAR_URL =
  "https://www.gravatar.com/avatar/00000000000000000000000000000000?d=blank&size=200";

const TAGGED_TEMPLATES = {
  liveQueryActivityTemplate: (
    activity: IActivity,
    onDetailsClick?: (details: IActivityDetails) => void
  ) => {
    const count = activity.details?.targets_count;
    const queryName = activity.details?.query_name;
    const querySql = activity.details?.query_sql;

    const savedQueryName = queryName ? (
      <>
        the <b>{queryName}</b> query as
      </>
    ) : (
      <></>
    );

    const hostCount =
      count !== undefined
        ? ` on ${count} ${count === 1 ? "host" : "hosts"}`
        : "";

    return (
      <>
        <span>
          ran {savedQueryName} a live query {hostCount}.
        </span>
        {querySql && (
          <>
            <Button
              className={`${baseClass}__show-query-link`}
              variant="text-link"
              onClick={() => onDetailsClick?.({ query_sql: querySql })}
            >
              Show query{" "}
              <Icon className={`${baseClass}__show-query-icon`} name="eye" />
            </Button>
          </>
        )}
      </>
    );
  },
  editPackCtlActivityTemplate: () => {
    return "edited a pack using fleetctl.";
  },
  editPolicyCtlActivityTemplate: () => {
    return "edited policies using fleetctl.";
  },
  editQueryCtlActivityTemplate: (activity: IActivity) => {
    const count = activity.details?.specs?.length;
    return typeof count === "undefined" || count === 1
      ? "edited a query using fleetctl."
      : "edited queries using fleetctl.";
  },
  editTeamCtlActivityTemplate: (activity: IActivity) => {
    const count = activity.details?.teams?.length;
    return count === 1 && activity.details?.teams ? (
      <>
        edited <b>{activity.details?.teams[0].name}</b> team using fleetctl.
      </>
    ) : (
      "edited multiple teams using fleetctl."
    );
  },
  userAddedBySSOTempalte: () => {
    return "was added to Fleet by SSO.";
  },
  editAgentOptions: (activity: IActivity) => {
    return activity.details?.global ? (
      "edited agent options."
    ) : (
      <>
        edited agent options on <b>{activity.details?.team_name}</b> team.
      </>
    );
  },

  defaultActivityTemplate: (activity: IActivity) => {
    const entityName = find(activity.details, (_, key) =>
      key.includes("_name")
    );

    const activityType = lowerCase(activity.type).replace(" saved", "");

    return !entityName ? (
      `${activityType}.`
    ) : (
      <span>
        {activityType} <b>{entityName}</b>.
      </span>
    );
  },
};

const getDetail = (
  activity: IActivity,
  onDetailsClick?: (details: IActivityDetails) => void
) => {
  switch (activity.type) {
    case ActivityType.LiveQuery: {
      return TAGGED_TEMPLATES.liveQueryActivityTemplate(
        activity,
        onDetailsClick
      );
    }
    case ActivityType.AppliedSpecPack: {
      return TAGGED_TEMPLATES.editPackCtlActivityTemplate();
    }
    case ActivityType.AppliedSpecPolicy: {
      return TAGGED_TEMPLATES.editPolicyCtlActivityTemplate();
    }
    case ActivityType.AppliedSpecSavedQuery: {
      return TAGGED_TEMPLATES.editQueryCtlActivityTemplate(activity);
    }
    case ActivityType.AppliedSpecTeam: {
      return TAGGED_TEMPLATES.editTeamCtlActivityTemplate(activity);
    }
    case ActivityType.UserAddedBySSO: {
      return TAGGED_TEMPLATES.userAddedBySSOTempalte();
    }
    case ActivityType.EditedAgentOptions: {
      return TAGGED_TEMPLATES.editAgentOptions(activity);
    }
    default: {
      return TAGGED_TEMPLATES.defaultActivityTemplate(activity);
    }
  }
};

interface IActivityItemProps {
  activity: IActivity;

  /** A handler for handling clicking on the details of an activity. Not all
   * activites have more details so this is optional. An example of additonal
   * details is showing the query for a live query action.
   */
  onDetailsClick?: (details: IActivityDetails) => void;
}

const ActivityItem = ({
  activity,
  onDetailsClick = noop,
}: IActivityItemProps) => {
  const { actor_email } = activity;
  const { gravatarURL } = actor_email
    ? addGravatarUrlToResource({ email: actor_email })
    : { gravatarURL: DEFAULT_GRAVATAR_URL };

  return (
    <div className={baseClass}>
      <Avatar
        className={`${baseClass}__avatar-image`}
        user={{ gravatarURL }}
        size="small"
      />
      <div className={`${baseClass}__details`}>
        <p>
          <span className={`${baseClass}__details-topline`}>
            <b>{activity.actor_full_name}</b>{" "}
            {getDetail(activity, onDetailsClick)}
          </span>
          <br />
          <span className={`${baseClass}__details-bottomline`}>
            {formatDistanceToNowStrict(new Date(activity.created_at), {
              addSuffix: true,
            })}
          </span>
        </p>
      </div>
    </div>
  );
};

export default ActivityItem;

import React from "react";
import { find, lowerCase, noop } from "lodash";
import { intlFormat, formatDistanceToNowStrict } from "date-fns";

import { ActivityType, IActivity, IActivityDetails } from "interfaces/activity";
import { addGravatarUrlToResource } from "utilities/helpers";
import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";
import Avatar from "components/Avatar";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import ReactTooltip from "react-tooltip";
import { actions } from "react-table";

const baseClass = "activity-item";

const getProfileMessageSuffix = (
  isPremiumTier: boolean,
  teamName?: string | null
) => {
  let messageSuffix = <>all macOS hosts</>;
  if (isPremiumTier) {
    messageSuffix = teamName ? (
      <>
        macOS hosts assigned to the <b>{teamName}</b> team
      </>
    ) : (
      <>macOS hosts with no team</>
    );
  }
  return messageSuffix;
};

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
  editAgentOptions: (activity: IActivity) => {
    return activity.details?.global ? (
      "edited agent options."
    ) : (
      <>
        edited agent options on <b>{activity.details?.team_name}</b> team.
      </>
    );
  },
  userAddedBySSOTempalte: () => {
    return "was added to Fleet by SSO.";
  },
  userLoggedIn: (activity: IActivity) => {
    return `successfully logged in from public IP ${activity.details?.public_ip}.`;
  },
  userFailedLogin: (activity: IActivity) => {
    return (
      <>
        Somebody using <b>{activity.details?.email}</b> failed to log in from
        public IP {activity.details?.public_ip}.
      </>
    );
  },
  userCreated: (activity: IActivity) => {
    return (
      <>
        created a user <b> {activity.details?.user_email}</b>.
      </>
    );
  },
  userDeleted: (activity: IActivity) => {
    return (
      <>
        deleted a user <b>{activity.details?.user_email}</b>.
      </>
    );
  },
  userChangedGlobalRole: (activity: IActivity, isPremiumTier: boolean) => {
    return (
      <>
        changed <b>{activity.details?.user_email}</b> to{" "}
        <b>{activity.details?.role}</b>
        {isPremiumTier && " for all teams"}.
      </>
    );
  },
  userDeletedGlobalRole: (activity: IActivity, isPremiumTier: boolean) => {
    return (
      <>
        removed <b>{activity.details?.user_email}</b> as{" "}
        <b>{activity.details?.role}</b>
        {isPremiumTier && " for all teams"}.
      </>
    );
  },
  userChangedTeamRole: (activity: IActivity) => {
    return (
      <>
        changed <b>{activity.details?.user_email}</b> to{" "}
        <b>{activity.details?.role}</b> for the{" "}
        <b>{activity.details?.team_name}</b> team.
      </>
    );
  },
  userDeletedTeamRole: (activity: IActivity) => {
    return (
      <>
        removed <b>{activity.details?.user_email}</b> from the{" "}
        <b>{activity.details?.team_name}</b> team.
      </>
    );
  },
  mdmEnrolled: (activity: IActivity) => {
    return (
      <>
        An end user turned on MDM features for a host with serial number{" "}
        <b>
          {activity.details?.host_serial} (
          {activity.details?.installed_from_dep ? "automatic" : "manual"})
        </b>
        .
      </>
    );
  },
  mdmUnenrolled: (activity: IActivity) => {
    return (
      <>
        {activity.actor_full_name
          ? " told Fleet to turn off mobile device management (MDM) for"
          : "Mobile device management (MDM) was turned off for"}{" "}
        <b>{activity.details?.host_display_name}</b>.
      </>
    );
  },
  editedMacosMinVersion: (activity: IActivity) => {
    const editedActivity =
      activity.details?.minimum_version === "" ? "removed" : "updated";

    const versionSection = activity.details?.minimum_version ? (
      <>
        to <b>{activity.details.minimum_version}</b>
      </>
    ) : null;

    const deadlineSection = activity.details?.deadline ? (
      <>(deadline: {activity.details.deadline})</>
    ) : null;

    const teamSection = activity.details?.team_id ? (
      <>
        the <b>{activity.details.team_name}</b> team
      </>
    ) : (
      <>no team</>
    );

    return (
      <>
        {editedActivity} the minimum macOS version {versionSection}{" "}
        {deadlineSection} on hosts assigned to {teamSection}.
      </>
    );
  },

  readHostDiskEncryptionKey: (activity: IActivity) => {
    return (
      <>
        {" "}
        viewed the disk encryption key for {activity.details?.host_display_name}
        .
      </>
    );
  },

  createMacOSProfile: (activity: IActivity, isPremiumTier: boolean) => {
    return (
      <>
        {" "}
        added configuration profile {activity.details?.profile_name} to{" "}
        {getProfileMessageSuffix(isPremiumTier, activity.details?.team_name)}.
      </>
    );
  },

  deleteMacOSProfile: (activity: IActivity, isPremiumTier: boolean) => {
    return (
      <>
        {" "}
        deleted configuration profile {
          activity.details?.host_display_name
        } from{" "}
        {getProfileMessageSuffix(isPremiumTier, activity.details?.team_name)}.
      </>
    );
  },

  editMacOSProfile: (activity: IActivity, isPremiumTier: boolean) => {
    return (
      <>
        {" "}
        edited configuration profiles for{" "}
        {getProfileMessageSuffix(
          isPremiumTier,
          activity.details?.team_name
        )}{" "}
        via fleetctl.
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
      <>
        {activityType} <b>{entityName}</b>.
      </>
    );
  },
};

const getDetail = (
  activity: IActivity,
  isPremiumTier: boolean,
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
    case ActivityType.EditedAgentOptions: {
      return TAGGED_TEMPLATES.editAgentOptions(activity);
    }
    case ActivityType.UserAddedBySSO: {
      return TAGGED_TEMPLATES.userAddedBySSOTempalte();
    }
    case ActivityType.UserLoggedIn: {
      return TAGGED_TEMPLATES.userLoggedIn(activity);
    }
    case ActivityType.UserFailedLogin: {
      return TAGGED_TEMPLATES.userFailedLogin(activity);
    }
    case ActivityType.UserCreated: {
      return TAGGED_TEMPLATES.userCreated(activity);
    }
    case ActivityType.UserDeleted: {
      return TAGGED_TEMPLATES.userDeleted(activity);
    }
    case ActivityType.UserChangedGlobalRole: {
      return TAGGED_TEMPLATES.userChangedGlobalRole(activity, isPremiumTier);
    }
    case ActivityType.UserDeletedGlobalRole: {
      return TAGGED_TEMPLATES.userDeletedGlobalRole(activity, isPremiumTier);
    }
    case ActivityType.UserChangedTeamRole: {
      return TAGGED_TEMPLATES.userChangedTeamRole(activity);
    }
    case ActivityType.UserDeletedTeamRole: {
      return TAGGED_TEMPLATES.userDeletedTeamRole(activity);
    }
    case ActivityType.MdmEnrolled: {
      return TAGGED_TEMPLATES.mdmEnrolled(activity);
    }
    case ActivityType.MdmUnenrolled: {
      return TAGGED_TEMPLATES.mdmUnenrolled(activity);
    }
    case ActivityType.EditedMacosMinVersion: {
      return TAGGED_TEMPLATES.editedMacosMinVersion(activity);
    }
    case ActivityType.ReadHostDiskEncryptionKey: {
      return TAGGED_TEMPLATES.readHostDiskEncryptionKey(activity);
    }
    case ActivityType.CreatedMacOSProfile: {
      return TAGGED_TEMPLATES.createMacOSProfile(activity, isPremiumTier);
    }
    case ActivityType.DeletedMacOSProfile: {
      return TAGGED_TEMPLATES.createMacOSProfile(activity, isPremiumTier);
    }
    case ActivityType.EditedMacOSProfile: {
      return TAGGED_TEMPLATES.createMacOSProfile(activity, isPremiumTier);
    }
    default: {
      return TAGGED_TEMPLATES.defaultActivityTemplate(activity);
    }
  }
};

interface IActivityItemProps {
  activity: IActivity;
  isPremiumTier: boolean;

  /** A handler for handling clicking on the details of an activity. Not all
   * activites have more details so this is optional. An example of additonal
   * details is showing the query for a live query action.
   */
  onDetailsClick?: (details: IActivityDetails) => void;
}

const ActivityItem = ({
  activity,
  isPremiumTier,
  onDetailsClick = noop,
}: IActivityItemProps) => {
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
      <div className={`${baseClass}__details`}>
        <p>
          <span className={`${baseClass}__details-topline`}>
            {activity.type === ActivityType.UserLoggedIn ? (
              <b>{activity.actor_email} </b>
            ) : (
              <b>{activity.actor_full_name} </b>
            )}
            {getDetail(activity, isPremiumTier, onDetailsClick)}
          </span>
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
            backgroundColor="#3e4771"
          >
            {intlFormat(
              activityCreatedAt,
              {
                year: "numeric",
                month: "numeric",
                day: "numeric",
                hour: "numeric",
                minute: "numeric",
                second: "numeric",
              },
              { locale: window.navigator.languages[0] }
            )}
          </ReactTooltip>
        </p>
      </div>
    </div>
  );
};

export default ActivityItem;

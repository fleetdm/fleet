import React from "react";
import { find, lowerCase, noop } from "lodash";
import { formatDistanceToNowStrict } from "date-fns";

import { ActivityType, IActivity, IActivityDetails } from "interfaces/activity";
import {
  addGravatarUrlToResource,
  formatScriptNameForActivityItem,
  getPerformanceImpactDescription,
  internationalTimeFormat,
} from "utilities/helpers";
import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";
import Avatar from "components/Avatar";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import ReactTooltip from "react-tooltip";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";
import { COLORS } from "styles/var/colors";

const baseClass = "activity-item";

const PREMIUM_ACTIVITIES = new Set([
  "created_team",
  "deleted_team",
  "applied_spec_team",
  "changed_user_team_role",
  "deleted_user_team_role",
  "read_host_disk_encryption_key",
  "enabled_macos_disk_encryption",
  "disabled_macos_disk_encryption",
  "enabled_macos_setup_end_user_auth",
  "disabled_macos_setup_end_user_auth",
  "tranferred_hosts",
]);

const getProfileMessageSuffix = (
  isPremiumTier: boolean,
  platform: "darwin" | "windows",
  teamName?: string | null
) => {
  const platformDisplayName = platform === "darwin" ? "macOS" : "Windows";
  let messageSuffix = <>all {platformDisplayName} hosts</>;
  if (isPremiumTier) {
    messageSuffix = teamName ? (
      <>
        {platformDisplayName} hosts assigned to the <b>{teamName}</b> team
      </>
    ) : (
      <>{platformDisplayName} hosts with no team</>
    );
  }
  return messageSuffix;
};

const getDiskEncryptionMessageSuffix = (teamName?: string | null) => {
  return teamName ? (
    <>
      {" "}
      assigned to the <b>{teamName}</b> team
    </>
  ) : (
    <>with no team</>
  );
};

const getMacOSSetupAssistantMessage = (
  action: "added" | "deleted",
  name?: string,
  teamName?: string | null
) => {
  const suffix = teamName ? (
    <>
      {" "}
      that automatically enroll to the <b>{teamName}</b> team
    </>
  ) : (
    <>that automatically enroll to no team</>
  );

  return (
    <>
      {" "}
      changed the macOS Setup Assistant ({action} <b>{name}</b>) for hosts{" "}
      {suffix}.
    </>
  );
};

const TAGGED_TEMPLATES = {
  liveQueryActivityTemplate: (
    activity: IActivity,
    onDetailsClick?: (type: ActivityType, details: IActivityDetails) => void
  ) => {
    const {
      targets_count: count,
      query_name: queryName,
      query_sql: querySql,
      stats,
    } = activity.details || {};

    const impactDescription = stats
      ? getPerformanceImpactDescription(stats)
      : undefined;

    const queryNameCopy = queryName ? (
      <>
        the <b>{queryName}</b> query
      </>
    ) : (
      <>a live query</>
    );

    const impactCopy =
      impactDescription && impactDescription !== "Undetermined" ? (
        <>with {impactDescription.toLowerCase()} performance impact</>
      ) : (
        <></>
      );
    const hostCountCopy =
      count !== undefined
        ? ` on ${count} ${count === 1 ? "host" : "hosts"}`
        : "";

    return (
      <>
        <span className={`${baseClass}__details-content`}>
          ran {queryNameCopy} {impactCopy} {hostCountCopy}.
        </span>
        {querySql && (
          <>
            <Button
              className={`${baseClass}__show-query-link`}
              variant="text-link"
              onClick={() =>
                onDetailsClick?.(ActivityType.LiveQuery, {
                  query_sql: querySql,
                  stats,
                })
              }
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
        edited the <b>{activity.details?.teams[0].name}</b> team using fleetctl.
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
    return activity.actor_id === activity.details?.user_id ? (
      <>activated their account.</>
    ) : (
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
    const { actor_id } = activity;
    const { user_id, user_email, role } = activity.details || {};

    if (actor_id === user_id) {
      // this is the case when SSO user is crated via JIT provisioning
      // should only be possible for premium tier, but check anyway
      return (
        <>
          was assigned the <b>{role}</b> role{isPremiumTier && " for all teams"}
          .
        </>
      );
    }
    return (
      <>
        changed <b>{user_email}</b> to <b>{activity.details?.role}</b>
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
    const { actor_id } = activity;
    const { user_id, user_email, role, team_name } = activity.details || {};

    const varText =
      actor_id === user_id ? (
        <>
          was assigned the <b>{role}</b> role
        </>
      ) : (
        <>
          changed <b>{user_email}</b> to <b>{role}</b>
        </>
      );
    return (
      <>
        {varText} for the <b>{team_name}</b> team.
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
    if (activity.details?.mdm_platform === "microsoft") {
      return (
        <>
          Mobile device management (MDM) was turned on for{" "}
          <b>{activity.details?.host_display_name} (manual)</b>.
        </>
      );
    }

    // note: if mdm_platform is missing, we assume this is Apple MDM for backwards
    // compatibility
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
        viewed the disk encryption key for{" "}
        <b>{activity.details?.host_display_name}</b>.
      </>
    );
  },
  createMacOSProfile: (activity: IActivity, isPremiumTier: boolean) => {
    const profileName = activity.details?.profile_name;
    return (
      <>
        {" "}
        added{" "}
        {profileName ? (
          <>
            configuration profile <b>{profileName}</b>
          </>
        ) : (
          <>a configuration profile</>
        )}{" "}
        to{" "}
        {getProfileMessageSuffix(
          isPremiumTier,
          "darwin",
          activity.details?.team_name
        )}
        .
      </>
    );
  },
  deleteMacOSProfile: (activity: IActivity, isPremiumTier: boolean) => {
    const profileName = activity.details?.profile_name;
    return (
      <>
        {" "}
        deleted{" "}
        {profileName ? (
          <>
            configuration profile <b>{profileName}</b>
          </>
        ) : (
          <>a configuration profile</>
        )}{" "}
        from{" "}
        {getProfileMessageSuffix(
          isPremiumTier,
          "darwin",
          activity.details?.team_name
        )}
        .
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
          "darwin",
          activity.details?.team_name
        )}{" "}
        via fleetctl.
      </>
    );
  },
  createWindowsProfile: (activity: IActivity, isPremiumTier: boolean) => {
    const profileName = activity.details?.profile_name;
    return (
      <>
        {" "}
        added{" "}
        {profileName ? (
          <>
            configuration profile <b>{profileName}</b>
          </>
        ) : (
          <>a configuration profile</>
        )}{" "}
        to{" "}
        {getProfileMessageSuffix(
          isPremiumTier,
          "windows",
          activity.details?.team_name
        )}
        .
      </>
    );
  },
  deleteWindowsProfile: (activity: IActivity, isPremiumTier: boolean) => {
    const profileName = activity.details?.profile_name;
    return (
      <>
        {" "}
        deleted{" "}
        {profileName ? (
          <>
            configuration profile <b>{profileName}</b>
          </>
        ) : (
          <>a configuration profile</>
        )}{" "}
        from{" "}
        {getProfileMessageSuffix(
          isPremiumTier,
          "windows",
          activity.details?.team_name
        )}
        .
      </>
    );
  },
  editWindowsProfile: (activity: IActivity, isPremiumTier: boolean) => {
    return (
      <>
        {" "}
        edited configuration profiles for{" "}
        {getProfileMessageSuffix(
          isPremiumTier,
          "windows",
          activity.details?.team_name
        )}{" "}
        via fleetctl.
      </>
    );
  },
  enabledDiskEncryption: (activity: IActivity) => {
    const suffix = getDiskEncryptionMessageSuffix(activity.details?.team_name);
    return <> enforced disk encryption for hosts {suffix}.</>;
  },
  disabledEncryption: (activity: IActivity) => {
    const suffix = getDiskEncryptionMessageSuffix(activity.details?.team_name);
    return <>removed disk encryption enforcement for hosts {suffix}.</>;
  },
  changedMacOSSetupAssistant: (activity: IActivity) => {
    return getMacOSSetupAssistantMessage(
      "added",
      activity.details?.name,
      activity.details?.team_name
    );
  },
  deletedMacOSSetupAssistant: (activity: IActivity) => {
    return getMacOSSetupAssistantMessage(
      "deleted",
      activity.details?.name,
      activity.details?.team_name
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
  addedMDMBootstrapPackage: (activity: IActivity) => {
    const packageName = activity.details?.bootstrap_package_name;
    return (
      <>
        {" "}
        added a bootstrap package{" "}
        {packageName ? (
          <>
            &#40;<b>{packageName}</b>&#41;{" "}
          </>
        ) : (
          ""
        )}
        for macOS hosts that automatically enroll to{" "}
        {activity.details?.team_name ? (
          <>
            the <b>{activity.details.team_name}</b> team
          </>
        ) : (
          "no team"
        )}
        .
      </>
    );
  },
  deletedMDMBootstrapPackage: (activity: IActivity) => {
    const packageName = activity.details?.bootstrap_package_name;
    return (
      <>
        {" "}
        deleted a bootstrap package{" "}
        {packageName ? (
          <>
            &#40;<b>{packageName}</b>&#41;{" "}
          </>
        ) : (
          ""
        )}
        for macOS hosts that automatically enroll to{" "}
        {activity.details?.team_name ? (
          <>
            the <b>{activity.details.team_name}</b> team
          </>
        ) : (
          "no team"
        )}
        .
      </>
    );
  },
  enabledMacOSSetupEndUserAuth: (activity: IActivity) => {
    return (
      <>
        {" "}
        required end user authentication for macOS hosts that automatically
        enroll to{" "}
        {activity.details?.team_name ? (
          <>
            the <b>{activity.details.team_name}</b> team
          </>
        ) : (
          "no team"
        )}
        .
      </>
    );
  },
  disabledMacOSSetupEndUserAuth: (activity: IActivity) => {
    return (
      <>
        {" "}
        removed end user authentication requirement for macOS hosts that
        automatically enroll to{" "}
        {activity.details?.team_name ? (
          <>
            the <b>{activity.details.team_name}</b> team
          </>
        ) : (
          "no team"
        )}
        .
      </>
    );
  },
  transferredHosts: (activity: IActivity) => {
    const hostNames = activity.details?.host_display_names || [];
    const teamName = activity.details?.team_name;
    if (hostNames.length === 1) {
      return (
        <>
          {" "}
          transferred host <b>{hostNames[0]}</b> to {teamName ? "team " : ""}
          <b>{teamName || "no team"}</b>.
        </>
      );
    }
    return (
      <>
        {" "}
        transferred {hostNames.length} hosts to {teamName ? "team " : ""}
        <b>{teamName || "no team"}</b>.
      </>
    );
  },

  enabledWindowsMdm: (activity: IActivity) => {
    return (
      <>
        {" "}
        told Fleet to turn on MDM features for all Windows hosts (servers
        excluded).
      </>
    );
  },
  disabledWindowsMdm: (activity: IActivity) => {
    return <> told Fleet to turn off Windows MDM features.</>;
  },
  // TODO: Combine ranScript template with host details page templates
  // frontend/pages/hosts/details/cards/Activity/PastActivity/PastActivity.tsx and
  // frontend/pages/hosts/details/cards/Activity/UpcomingActivity/UpcomingActivity.tsx
  ranScript: (
    activity: IActivity,
    onDetailsClick?: (type: ActivityType, details: IActivityDetails) => void
  ) => {
    const { script_name, host_display_name, script_execution_id } =
      activity.details || {};
    return (
      <>
        {" "}
        ran {formatScriptNameForActivityItem(script_name)} on{" "}
        {host_display_name}.{" "}
        <Button
          className={`${baseClass}__show-query-link`}
          variant="text-link"
          onClick={() =>
            onDetailsClick?.(ActivityType.RanScript, {
              script_execution_id,
            })
          }
        >
          Show details{" "}
          <Icon className={`${baseClass}__show-query-icon`} name="eye" />
        </Button>
      </>
    );
  },
  addedScript: (activity: IActivity) => {
    const scriptName = activity.details?.script_name;
    return (
      <>
        {" "}
        added{" "}
        {scriptName ? (
          <>
            script <b>{scriptName}</b>{" "}
          </>
        ) : (
          "a script "
        )}
        to{" "}
        {activity.details?.team_name ? (
          <>
            the <b>{activity.details.team_name}</b> team
          </>
        ) : (
          "no team"
        )}
        .
      </>
    );
  },
  deletedScript: (activity: IActivity) => {
    const scriptName = activity.details?.script_name;
    return (
      <>
        {" "}
        deleted{" "}
        {scriptName ? (
          <>
            script <b>{scriptName}</b>{" "}
          </>
        ) : (
          "a script "
        )}
        from{" "}
        {activity.details?.team_name ? (
          <>
            the <b>{activity.details.team_name}</b> team
          </>
        ) : (
          "no team"
        )}
        .
      </>
    );
  },
  editedScript: (activity: IActivity) => {
    return (
      <>
        {" "}
        edited scripts for{" "}
        {activity.details?.team_name ? (
          <>
            the <b>{activity.details.team_name}</b> team
          </>
        ) : (
          "no team"
        )}{" "}
        via fleetctl.
      </>
    );
  },
  editedWindowsUpdates: (activity: IActivity) => {
    return (
      <>
        {" "}
        updated the Windows OS update options (
        <b>
          Deadline: {activity.details?.deadline_days} days / Grace period:{" "}
          {activity.details?.grace_period_days} days
        </b>
        ) on hosts assigned to{" "}
        {activity.details?.team_name ? (
          <>
            the <b>{activity.details.team_name}</b> team
          </>
        ) : (
          "no team"
        )}
        .
      </>
    );
  },
  deletedMultipleSavedQuery: (activity: IActivity) => {
    return <> deleted multiple queries.</>;
  },
  lockedHost: (activity: IActivity) => {
    return (
      <>
        {" "}
        locked <b>{activity.details?.host_display_name}</b>.
      </>
    );
  },
  unlockedHost: (activity: IActivity) => {
    if (activity.details?.host_platform === "darwin") {
      return (
        <>
          {" "}
          viewed the six-digit unlock PIN for{" "}
          <b>{activity.details?.host_display_name}</b>.
        </>
      );
    }
    return (
      <>
        {" "}
        unlocked <b>{activity.details?.host_display_name}</b>.
      </>
    );
  },
};

const getDetail = (
  activity: IActivity,
  isPremiumTier: boolean,
  onDetailsClick?: (
    activityType: ActivityType,
    details: IActivityDetails
  ) => void
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
      return TAGGED_TEMPLATES.deleteMacOSProfile(activity, isPremiumTier);
    }
    case ActivityType.EditedMacOSProfile: {
      return TAGGED_TEMPLATES.editMacOSProfile(activity, isPremiumTier);
    }
    case ActivityType.CreatedWindowsProfile: {
      return TAGGED_TEMPLATES.createWindowsProfile(activity, isPremiumTier);
    }
    case ActivityType.DeletedWindowsProfile: {
      return TAGGED_TEMPLATES.deleteWindowsProfile(activity, isPremiumTier);
    }
    case ActivityType.EditedWindowsProfile: {
      return TAGGED_TEMPLATES.editWindowsProfile(activity, isPremiumTier);
    }
    // Note: Both "enabled_disk_encryption" and "enabled_macos_disk_encryption" display the same
    // message. The latter is deprecated in the API but it is retained here for backwards compatibility.
    case ActivityType.EnabledDiskEncryption:
    case ActivityType.EnabledMacDiskEncryption: {
      return TAGGED_TEMPLATES.enabledDiskEncryption(activity);
    }
    // Note: Both "disabled_disk_encryption" and "disabled_macos_disk_encryption" display the same
    // message. The latter is deprecated in the API but it is retained here for backwards compatibility.
    case ActivityType.DisabledDiskEncryption:
    case ActivityType.DisabledMacDiskEncryption: {
      return TAGGED_TEMPLATES.disabledEncryption(activity);
    }
    case ActivityType.AddedBootstrapPackage: {
      return TAGGED_TEMPLATES.addedMDMBootstrapPackage(activity);
    }
    case ActivityType.DeletedBootstrapPackage: {
      return TAGGED_TEMPLATES.deletedMDMBootstrapPackage(activity);
    }
    case ActivityType.ChangedMacOSSetupAssistant: {
      return TAGGED_TEMPLATES.changedMacOSSetupAssistant(activity);
    }
    case ActivityType.DeletedMacOSSetupAssistant: {
      return TAGGED_TEMPLATES.deletedMacOSSetupAssistant(activity);
    }
    case ActivityType.EnabledMacOSSetupEndUserAuth: {
      return TAGGED_TEMPLATES.enabledMacOSSetupEndUserAuth(activity);
    }
    case ActivityType.DisabledMacOSSetupEndUserAuth: {
      return TAGGED_TEMPLATES.disabledMacOSSetupEndUserAuth(activity);
    }
    case ActivityType.TransferredHosts: {
      return TAGGED_TEMPLATES.transferredHosts(activity);
    }
    case ActivityType.EnabledWindowsMdm: {
      return TAGGED_TEMPLATES.enabledWindowsMdm(activity);
    }
    case ActivityType.DisabledWindowsMdm: {
      return TAGGED_TEMPLATES.disabledWindowsMdm(activity);
    }
    case ActivityType.RanScript: {
      return TAGGED_TEMPLATES.ranScript(activity, onDetailsClick);
    }
    case ActivityType.AddedScript: {
      return TAGGED_TEMPLATES.addedScript(activity);
    }
    case ActivityType.DeletedScript: {
      return TAGGED_TEMPLATES.deletedScript(activity);
    }
    case ActivityType.EditedScript: {
      return TAGGED_TEMPLATES.editedScript(activity);
    }
    case ActivityType.EditedWindowsUpdates: {
      return TAGGED_TEMPLATES.editedWindowsUpdates(activity);
    }
    case ActivityType.DeletedMultipleSavedQuery: {
      return TAGGED_TEMPLATES.deletedMultipleSavedQuery(activity);
    }
    case ActivityType.LockedHost: {
      return TAGGED_TEMPLATES.lockedHost(activity);
    }
    case ActivityType.UnlockedHost: {
      return TAGGED_TEMPLATES.unlockedHost(activity);
    }
    default: {
      return TAGGED_TEMPLATES.defaultActivityTemplate(activity);
    }
  }
};

interface IActivityItemProps {
  activity: IActivity;
  isPremiumTier: boolean;
  isSandboxMode?: boolean;

  /** A handler for handling clicking on the details of an activity. Not all
   * activites have more details so this is optional. An example of additonal
   * details is showing the query for a live query action.
   */
  onDetailsClick?: (
    activityType: ActivityType,
    details: IActivityDetails
  ) => void;
}

const ActivityItem = ({
  activity,
  isPremiumTier,
  isSandboxMode = false,
  onDetailsClick = noop,
}: IActivityItemProps) => {
  const { actor_email } = activity;
  const { gravatar_url } = actor_email
    ? addGravatarUrlToResource({ email: actor_email })
    : { gravatar_url: DEFAULT_GRAVATAR_LINK };

  const activityCreatedAt = new Date(activity.created_at);
  const indicatePremiumFeature =
    isSandboxMode && PREMIUM_ACTIVITIES.has(activity.type);

  const renderActivityPrefix = () => {
    if (activity.type === ActivityType.UserLoggedIn) {
      return <b>{activity.actor_email} </b>;
    }
    if (
      (activity.type === ActivityType.UserChangedGlobalRole ||
        activity.type === ActivityType.UserChangedTeamRole) &&
      activity.actor_id === activity.details?.user_id
    ) {
      return <b>{activity.details?.user_email} </b>;
    }
    return <b>{activity.actor_full_name} </b>;
  };
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
          {indicatePremiumFeature && <PremiumFeatureIconWithTooltip />}
          <span className={`${baseClass}__details-topline`}>
            {renderActivityPrefix()}
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

export default ActivityItem;

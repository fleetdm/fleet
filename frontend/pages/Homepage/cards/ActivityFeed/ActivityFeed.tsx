import React, { useState } from "react";
import { useQuery } from "react-query";
import { find, isEmpty, lowerCase } from "lodash";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";

import activitiesAPI, {
  IActivitiesResponse,
} from "services/entities/activities";
import { addGravatarUrlToResource } from "utilities/helpers";

import { IActivity, ActivityType } from "interfaces/activity";

import DataError from "components/DataError";
import Avatar from "components/Avatar";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

const baseClass = "activity-feed";

interface IActvityCardProps {
  setShowActivityFeedTitle: (showActivityFeedTitle: boolean) => void;
}

interface IActivityDisplay extends IActivity {
  key?: string;
}

const DEFAULT_GRAVATAR_URL =
  "https://www.gravatar.com/avatar/00000000000000000000000000000000?d=blank&size=200";

const DEFAULT_PAGE_SIZE = 8;

const TAGGED_TEMPLATES = {
  liveQueryActivityTemplate: (activity: IActivity) => {
    const count = activity.details?.targets_count;
    return typeof count === "undefined" || typeof count !== "number"
      ? "ran a live query"
      : `ran a live query on ${count} ${count === 1 ? "host" : "hosts"}`;
  },
  editPackCtlActivityTemplate: () => {
    return "edited a pack using fleetctl";
  },
  editPolicyCtlActivityTemplate: () => {
    return "edited policies using fleetctl";
  },
  editQueryCtlActivityTemplate: (activity: IActivity) => {
    const count = activity.details?.specs?.length;
    return typeof count === "undefined" || typeof count !== "number"
      ? "edited a query using fleetctl"
      : `edited ${count === 1 ? "a query" : "queries"} using fleetctl`;
  },
  userAddedBySSOTempalte: () => {
    return `was added to Fleet by SSO`;
  },
  editAgentOptions: (activity: IActivity) => {
    return activity.details?.global ? (
      "edited agent options"
    ) : (
      <>
        edited agent options on <b>{activity.details?.team_name}</b> team
      </>
    );
  },

  defaultActivityTemplate: (activity: IActivity) => {
    const entityName = find(activity.details, (_, key) =>
      key.includes("_name")
    );

    const activityType = lowerCase(activity.type).replace(" saved", "");

    return !entityName ? (
      `${activityType}`
    ) : (
      <span>
        {activityType} <b>{entityName}</b>
      </span>
    );
  },
};

const ActivityFeed = ({
  setShowActivityFeedTitle,
}: IActvityCardProps): JSX.Element => {
  const [pageIndex, setPageIndex] = useState(0);
  const [showMore, setShowMore] = useState(true);

  const {
    data: activities,
    error: errorActivities,
    isFetching: isFetchingActivities,
  } = useQuery<
    IActivitiesResponse,
    Error,
    IActivity[],
    Array<{
      scope: string;
      pageIndex: number;
      perPage: number;
    }>
  >(
    [{ scope: "activities", pageIndex, perPage: DEFAULT_PAGE_SIZE }],
    ({ queryKey: [{ pageIndex: page, perPage }] }) => {
      return activitiesAPI.loadNext(page, perPage);
    },
    {
      keepPreviousData: true,
      staleTime: 5000,
      select: (data) => {
        // We purposly removed the "applied_spec_team" activity as we are currently
        // thinking how we want to display this in the UI.
        return data.activities.filter(
          (activity) => activity.type !== ActivityType.AppliedSpecTeam
        );
      },
      onSuccess: (results) => {
        setShowActivityFeedTitle(true);
        if (results.length < DEFAULT_PAGE_SIZE) {
          setShowMore(false);
        }
      },
      onError: () => {
        setShowActivityFeedTitle(true);
      },
    }
  );

  const onLoadPrevious = () => {
    setShowMore(true);
    setPageIndex(pageIndex - 1);
  };

  const onLoadNext = () => {
    setPageIndex(pageIndex + 1);
  };

  const getDetail = (activity: IActivity) => {
    switch (activity.type) {
      case ActivityType.LiveQuery: {
        return TAGGED_TEMPLATES.liveQueryActivityTemplate(activity);
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

  const renderError = () => {
    return <DataError card />;
  };

  const renderNoActivities = () => {
    return (
      <div className={`${baseClass}__no-activities`}>
        <p>
          <b>Fleet has not recorded any activity.</b>
        </p>
        <p>
          Try editing a query, updating your policies, or running a live query.
        </p>
      </div>
    );
  };

  const renderActivityBlock = (activity: IActivityDisplay) => {
    const { actor_email, id, key } = activity;
    const { gravatarURL } = actor_email
      ? addGravatarUrlToResource({ email: actor_email })
      : { gravatarURL: DEFAULT_GRAVATAR_URL };

    return (
      <div className={`${baseClass}__block`} key={key || id}>
        <Avatar
          className={`${baseClass}__avatar-image`}
          user={{
            gravatarURL,
          }}
          size="small"
        />
        <div className={`${baseClass}__details`}>
          <p className={`${baseClass}__details-topline`}>
            <b>{activity.actor_full_name}</b> {getDetail(activity)}.
          </p>
          <span className={`${baseClass}__details-bottomline`}>
            {formatDistanceToNowStrict(new Date(activity.created_at), {
              addSuffix: true,
            })}
          </span>
        </div>
      </div>
    );
  };

  // Renders opaque information as activity feed is loading
  const opacity = isFetchingActivities ? { opacity: 0.4 } : { opacity: 1 };

  return (
    <div className={baseClass}>
      {errorActivities && renderError()}
      {!errorActivities && !isFetchingActivities && isEmpty(activities) ? (
        renderNoActivities()
      ) : (
        <>
          {isFetchingActivities && (
            <div className="spinner">
              <Spinner />
            </div>
          )}
          <div style={opacity}>
            {activities?.map((activity) => renderActivityBlock(activity))}
          </div>
        </>
      )}
      {!errorActivities &&
        (!isEmpty(activities) || (isEmpty(activities) && pageIndex > 0)) && (
          <div className={`${baseClass}__pagination`}>
            <Button
              disabled={isFetchingActivities || pageIndex === 0}
              onClick={onLoadPrevious}
              variant="unstyled"
              className={`${baseClass}__load-activities-button`}
            >
              <>
                <FleetIcon name="chevronleft" /> Previous
              </>
            </Button>
            <Button
              disabled={isFetchingActivities || !showMore}
              onClick={onLoadNext}
              variant="unstyled"
              className={`${baseClass}__load-activities-button`}
            >
              <>
                Next <FleetIcon name="chevronright" />
              </>
            </Button>
          </div>
        )}
    </div>
  );
};

export default ActivityFeed;

import React, { useEffect, useState } from "react";
import { find, isEmpty, lowerCase } from "lodash";
import moment from "moment";

// @ts-ignore
// import Fleet from "fleet";
import activitiesAPI from "services/entities/activities";
import { addGravatarUrlToResource } from "fleet/helpers";

import { IActivity, ActivityType } from "interfaces/activity";

import Avatar from "components/Avatar";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner"; // @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

import ErrorIcon from "../../../../../assets/images/icon-error-16x16@2x.png";
import OpenNewTabIcon from "../../../../../assets/images/open-new-tab-12x12@2x.png";

const baseClass = "activity-feed";

const DEFAULT_GRAVATAR_URL =
  "https://www.gravatar.com/avatar/00000000000000000000000000000000?d=blank&size=200";

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
  editQueryCtlActivityTemplate: (activity: IActivity) => {
    const count = activity.details?.specs?.length;
    return typeof count === "undefined" || typeof count !== "number"
      ? "edited a query using fleetctl"
      : `edited ${count === 1 ? "a query" : "queries"} using fleetctl`;
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

const ActivityFeed = (): JSX.Element => {
  const [activities, setActivities] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingError, setIsLoadingError] = useState(false);
  const [pageIndex, setPageIndex] = useState(0);
  const [showMore, setShowMore] = useState(true);

  useEffect((): void => {
    const getActivities = async (): Promise<void> => {
      try {
        const { activities: responseActivities } = await activitiesAPI.loadNext(
          pageIndex
        );

        if (responseActivities.length) {
          setActivities(responseActivities);
        } else {
          setShowMore(false);
        }

        setIsLoading(false);
      } catch (err) {
        setIsLoadingError(true);
        setIsLoading(false);
      }
    };

    getActivities();
  }, [pageIndex]);

  const onLoadPrevious = () => {
    setIsLoading(true);
    setShowMore(true);
    setPageIndex(pageIndex - 1);
  };

  const onLoadNext = () => {
    setIsLoading(true);
    setPageIndex(pageIndex + 1);
  };

  const getDetail = (activity: IActivity) => {
    if (activity.type === ActivityType.LiveQuery) {
      return TAGGED_TEMPLATES.liveQueryActivityTemplate(activity);
    }
    if (activity.type === ActivityType.AppliedSpecPack) {
      return TAGGED_TEMPLATES.editPackCtlActivityTemplate();
    }
    if (activity.type === ActivityType.AppliedSpecSavedQuery) {
      return TAGGED_TEMPLATES.editQueryCtlActivityTemplate(activity);
    }
    return TAGGED_TEMPLATES.defaultActivityTemplate(activity);
  };

  const renderActivityBlock = (activity: IActivity, i: number) => {
    const { actor_email } = activity;
    const { gravatarURL } = actor_email
      ? addGravatarUrlToResource({ email: actor_email })
      : { gravatarURL: DEFAULT_GRAVATAR_URL };

    return (
      <div className={`${baseClass}__block`} key={i}>
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
            {moment(activity.created_at).fromNow()}
          </span>
        </div>
      </div>
    );
  };

  const renderError = () => {
    return (
      <div className={`${baseClass}__error`}>
        <div className={`${baseClass}__inner`}>
          <span className="info__header">
            <img src={ErrorIcon} alt="error icon" id="error-icon" />
            Something&apos;s gone wrong.
          </span>
          <span className="info__data">Refresh the page or log in again.</span>
          <span className="info__data">
            If this keeps happening, please&nbsp;
            <a
              href="https://github.com/fleetdm/fleet/issues"
              target="_blank"
              rel="noopener noreferrer"
            >
              file an issue
              <img src={OpenNewTabIcon} alt="open new tab" id="new-tab-icon" />
            </a>
          </span>
        </div>
      </div>
    );
  };

  const renderNoActivities = () => {
    return (
      <div className={`${baseClass}__no-activities`}>
        <p>
          <b>Fleet has not recorded any activities.</b>
        </p>
        <p>
          Did you recently edit your queries, update your packs, or run a live
          query? Try again in a few seconds as the system catches up.
        </p>
      </div>
    );
  };

  const renderActivities = activities.map((activity: IActivity, i: number) =>
    renderActivityBlock(activity, i)
  );

  return (
    <div className={baseClass}>
      {isLoadingError && renderError()}
      {!isLoadingError && !isLoading && isEmpty(activities) ? (
        renderNoActivities()
      ) : (
        <div>{renderActivities}</div>
      )}
      {isLoading && <Spinner />}
      {!isLoadingError && !isEmpty(activities) && (
        <div className={`${baseClass}__pagination`}>
          <Button
            disabled={isLoading || pageIndex === 0}
            onClick={onLoadPrevious}
            variant="unstyled"
            className={`${baseClass}__load-activities-button`}
          >
            <>
              <FleetIcon name="chevronleft" /> Previous
            </>
          </Button>
          <Button
            disabled={isLoading || !showMore}
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

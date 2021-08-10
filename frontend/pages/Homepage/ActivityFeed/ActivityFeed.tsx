import React, { useEffect, useState } from "react";
import { find, isEmpty, lowerCase } from "lodash";
import moment from "moment";

// @ts-ignore
import Fleet from "fleet";
import { addGravatarUrlToResource } from "fleet/helpers";

import { IActivity, ActivityType } from "interfaces/activity";

import Avatar from "components/Avatar";
import Button from "components/buttons/Button";
import Spinner from "components/loaders/Spinner";

import ErrorIcon from "../../../../assets/images/icon-error-16x16@2x.png";
import OpenNewTabIcon from "../../../../assets/images/open-new-tab-12x12@2x.png";

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
  defaultActivityTemplate: (activity: IActivity) => {
    const entityName = find(activity.details, (_, key) =>
      key.includes("_name")
    );
    return !entityName ? (
      `${lowerCase(activity.type)}`
    ) : (
      <span>
        {lowerCase(activity.type)} <b>{entityName}</b>
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
        const responseActivities = await Fleet.activities.loadNext(pageIndex);
        if (responseActivities.length) {
          setActivities((prevActivities) =>
            prevActivities.concat(responseActivities)
          );
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

  const onLoadMore = () => {
    setIsLoading(true);
    setPageIndex(pageIndex + 1);
  };

  const getDetail = (activity: IActivity) => {
    if (activity.type === ActivityType.LiveQuery) {
      return TAGGED_TEMPLATES.liveQueryActivityTemplate(activity);
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
        <img className="error-icon" alt="error icon" src={ErrorIcon} />
        <b>Something&rsquo;s gone wrong.</b>
        <p>Refresh the page or log in again.</p>
        <p>
          If this keeps happening, please{" "}
          <a
            href="https://github.com/fleetdm/fleet/issues"
            target="_blank"
            rel="noopener noreferrer"
          >
            file an issue
            <img src={OpenNewTabIcon} alt="open new tab" />
          </a>
        </p>
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
      {!isLoadingError && !isEmpty(activities) && showMore && (
        <Button
          disabled={isLoading}
          onClick={onLoadMore}
          variant="unstyled"
          className={`${baseClass}__load-more-button`}
        >
          Load more
        </Button>
      )}
      {!isLoadingError && !isLoading && !showMore && (
        <div className={`${baseClass}__no-more-activities`}>
          <p>You have no more recorded activity.</p>
        </div>
      )}
    </div>
  );
};

export default ActivityFeed;

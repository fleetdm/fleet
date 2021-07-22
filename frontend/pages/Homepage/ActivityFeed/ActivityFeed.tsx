import React, { useEffect, useState } from "react";
import moment from "moment";

// @ts-ignore
import Fleet from "fleet";
import { find, lowerCase } from "lodash";

import { IActivity, ActivityType } from "interfaces/activity";

import Avatar from "components/Avatar";
import Button from "components/buttons/Button";
import Spinner from "components/loaders/Spinner";

const baseClass = "activity-feed";

const DEFAULT_GRAVATAR_URL =
  "https://www.gravatar.com/avatar/00000000000000000000000000000000?d=blank&size=200";

const TAGGED_TEMPLATES = {
  liveQueryActivityTemplate: (activity: IActivity) => {
    const count = activity.details?.targets_count;
    return typeof count === undefined
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
  const [pageIndex, setPageIndex] = useState(0);
  const [showMore, setShowMore] = useState(true);

  useEffect((): void => {
    const getActivities = async (): Promise<void> => {
      const newItems = await Fleet.activities.loadNext(pageIndex);
      if (newItems.length) {
        setActivities((prevItems) => prevItems.concat(newItems));
      } else {
        setShowMore(false);
      }
      setIsLoading(false);
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
    const gravatarURL = activity.actor_gravatar || DEFAULT_GRAVATAR_URL;
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

  return (
    <div className={baseClass}>
      {activities &&
        activities.map((activity: IActivity, i: number) =>
          renderActivityBlock(activity, i)
        )}
      {isLoading && <Spinner />}
      {showMore && (
        <Button
          disabled={isLoading}
          onClick={onLoadMore}
          variant="unstyled"
          className={`${baseClass}__load-more-button`}
        >
          Load more
        </Button>
      )}
    </div>
  );
};

export default ActivityFeed;

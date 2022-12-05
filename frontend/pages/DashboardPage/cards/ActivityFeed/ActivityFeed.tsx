import React, { useRef, useState } from "react";
import { useQuery } from "react-query";
import { isEmpty } from "lodash";

import activitiesAPI, {
  IActivitiesResponse,
} from "services/entities/activities";

import { IActivity, IActivityDetails } from "interfaces/activity";

import ShowQueryModal from "components/modals/ShowQueryModal";
import DataError from "components/DataError";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import ActivityItem from "./ActivityItem";

const baseClass = "activity-feed";
interface IActvityCardProps {
  setShowActivityFeedTitle: (showActivityFeedTitle: boolean) => void;
}

const DEFAULT_PAGE_SIZE = 8;

const ActivityFeed = ({
  setShowActivityFeedTitle,
}: IActvityCardProps): JSX.Element => {
  const [pageIndex, setPageIndex] = useState(0);
  const [showMore, setShowMore] = useState(true);
  const [showShowQueryModal, setShowShowQueryModal] = useState(false);
  const queryShown = useRef("");

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
        return data.activities;
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

  const handleDetailsClick = (details: IActivityDetails) => {
    queryShown.current = details.query_sql ?? "";
    setShowShowQueryModal(true);
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
            {activities?.map((activity) => (
              <ActivityItem
                activity={activity}
                onDetailsClick={handleDetailsClick}
                key={activity.id}
              />
            ))}
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
      {showShowQueryModal && (
        <ShowQueryModal
          query={queryShown.current}
          onCancel={() => setShowShowQueryModal(false)}
        />
      )}
    </div>
  );
};

export default ActivityFeed;

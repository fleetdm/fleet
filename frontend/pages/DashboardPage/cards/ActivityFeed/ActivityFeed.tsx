import React, { useRef, useState } from "react";
import { useQuery } from "react-query";
import { isEmpty } from "lodash";

import activitiesAPI, {
  IActivitiesResponse,
} from "services/entities/activities";

import { ActivityType, IActivityDetails } from "interfaces/activity";
import { getPerformanceImpactDescription } from "utilities/helpers";

import ShowQueryModal from "components/modals/ShowQueryModal";
import DataError from "components/DataError";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

import { SoftwareInstallDetailsModal } from "pages/SoftwarePage/components/SoftwareInstallDetails";

import ActivityItem from "./ActivityItem";
import ScriptDetailsModal from "./components/ScriptDetailsModal/ScriptDetailsModal";

const baseClass = "activity-feed";
interface IActvityCardProps {
  setShowActivityFeedTitle: (showActivityFeedTitle: boolean) => void;
  isPremiumTier: boolean;
}

const DEFAULT_PAGE_SIZE = 8;

const ActivityFeed = ({
  setShowActivityFeedTitle,
  isPremiumTier,
}: IActvityCardProps): JSX.Element => {
  const [pageIndex, setPageIndex] = useState(0);
  const [showShowQueryModal, setShowShowQueryModal] = useState(false);
  const [showScriptDetailsModal, setShowScriptDetailsModal] = useState(false);
  const [installedSoftwareUuid, setInstalledSoftwareUuid] = useState("");
  const queryShown = useRef("");
  const queryImpact = useRef<string | undefined>(undefined);
  const scriptExecutionId = useRef("");

  const {
    data: activitiesData,
    error: errorActivities,
    isFetching: isFetchingActivities,
  } = useQuery<
    IActivitiesResponse,
    Error,
    IActivitiesResponse,
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
      onSuccess: () => {
        setShowActivityFeedTitle(true);
      },
      onError: () => {
        setShowActivityFeedTitle(true);
      },
    }
  );

  const onLoadPrevious = () => {
    setPageIndex(pageIndex - 1);
  };

  const onLoadNext = () => {
    setPageIndex(pageIndex + 1);
  };

  const handleDetailsClick = (
    activityType: ActivityType,
    details: IActivityDetails
  ) => {
    console.log("activityType", activityType);
    switch (activityType) {
      case ActivityType.LiveQuery:
        queryShown.current = details.query_sql ?? "";
        queryImpact.current = details.stats
          ? getPerformanceImpactDescription(details.stats)
          : undefined;
        setShowShowQueryModal(true);
        break;
      case ActivityType.RanScript:
        scriptExecutionId.current = details.script_execution_id ?? "";
        setShowScriptDetailsModal(true);
        break;
      case ActivityType.InstalledSoftware:
        // installUuid.current = details.install_uuid ?? "";
        // console.log("installUuid.current", installUuid.current);
        setInstalledSoftwareUuid(details.install_uuid ?? "");
        break;
      default:
        break;
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

  // Renders opaque information as activity feed is loading
  const opacity = isFetchingActivities ? { opacity: 0.4 } : { opacity: 1 };

  const activities = activitiesData?.activities;
  const meta = activitiesData?.meta;

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
                isPremiumTier={isPremiumTier}
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
              disabled={isFetchingActivities || !meta?.has_previous_results}
              onClick={onLoadPrevious}
              variant="unstyled"
              className={`${baseClass}__load-activities-button`}
            >
              <>
                <FleetIcon name="chevronleft" /> Previous
              </>
            </Button>
            <Button
              disabled={isFetchingActivities || !meta?.has_next_results}
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
          impact={queryImpact.current}
          onCancel={() => setShowShowQueryModal(false)}
        />
      )}
      {showScriptDetailsModal && (
        <ScriptDetailsModal
          scriptExecutionId={scriptExecutionId.current}
          onCancel={() => setShowScriptDetailsModal(false)}
        />
      )}
      {installedSoftwareUuid && (
        <SoftwareInstallDetailsModal
          installUuid={installedSoftwareUuid}
          onCancel={() => setInstalledSoftwareUuid("")}
        />
      )}
    </div>
  );
};

export default ActivityFeed;

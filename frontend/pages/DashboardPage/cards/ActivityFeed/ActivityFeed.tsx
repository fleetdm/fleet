import React, { useRef, useState } from "react";
import { useQuery } from "react-query";
import { isEmpty } from "lodash";
import { InjectedRouter } from "react-router";

import activitiesAPI, {
  IActivitiesResponse,
} from "services/entities/activities";

import { ActivityType, IActivityDetails } from "interfaces/activity";
import { getPerformanceImpactDescription } from "utilities/helpers";

import ShowQueryModal from "components/modals/ShowQueryModal";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import Pagination from "components/Pagination";

import { AppInstallDetailsModal } from "components/ActivityDetails/InstallDetails/AppInstallDetails";
import { SoftwareInstallDetailsModal } from "components/ActivityDetails/InstallDetails/SoftwareInstallDetailsModal/SoftwareInstallDetailsModal";
import SoftwareUninstallDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";
import { IShowActivityDetailsData } from "components/ActivityItem/ActivityItem";

import GlobalActivityItem from "./GlobalActivityItem";
import ActivityAutomationDetailsModal from "./components/ActivityAutomationDetailsModal";
import RunScriptDetailsModal from "./components/RunScriptDetailsModal/RunScriptDetailsModal";
import SoftwareDetailsModal from "./components/LibrarySoftwareDetailsModal";
import VppDetailsModal from "./components/VPPDetailsModal";
import ScriptBatchSummaryModal from "./components/ScriptBatchSummaryModal";

const baseClass = "activity-feed";
interface IActvityCardProps {
  setShowActivityFeedTitle: (showActivityFeedTitle: boolean) => void;
  setRefetchActivities: (refetch: () => void) => void;
  isPremiumTier: boolean;
  router: InjectedRouter;
}

const DEFAULT_PAGE_SIZE = 8;

const ActivityFeed = ({
  setShowActivityFeedTitle,
  setRefetchActivities,
  isPremiumTier,
  router,
}: IActvityCardProps): JSX.Element => {
  const [pageIndex, setPageIndex] = useState(0);
  const [showShowQueryModal, setShowShowQueryModal] = useState(false);
  const [showScriptDetailsModal, setShowScriptDetailsModal] = useState(false);
  const [
    packageInstallDetails,
    setPackageInstallDetails,
  ] = useState<IActivityDetails | null>(null);
  const [
    packageUninstallDetails,
    setPackageUninstallDetails,
  ] = useState<IActivityDetails | null>(null);
  const [
    appInstallDetails,
    setAppInstallDetails,
  ] = useState<IActivityDetails | null>(null);
  const [
    activityAutomationDetails,
    setActivityAutomationDetails,
  ] = useState<IActivityDetails | null>(null);
  const [
    softwareDetails,
    setSoftwareDetails,
  ] = useState<IActivityDetails | null>(null);
  const [vppDetails, setVppDetails] = useState<IActivityDetails | null>(null);
  const [
    scriptBatchExecutionDetails,
    setScriptBatchExecutionDetails,
  ] = useState<IActivityDetails | null>(null);

  const queryShown = useRef("");
  const queryImpact = useRef<string | undefined>(undefined);
  const scriptExecutionId = useRef("");

  const {
    data: activitiesData,
    error: errorActivities,
    isFetching: isFetchingActivities,
    refetch,
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

  setRefetchActivities(refetch);

  const onLoadPrevious = () => {
    setPageIndex(pageIndex - 1);
  };

  const onLoadNext = () => {
    setPageIndex(pageIndex + 1);
  };

  const handleDetailsClick = ({
    type,
    details,
    created_at,
  }: IShowActivityDetailsData) => {
    switch (type) {
      case ActivityType.LiveQuery:
        queryShown.current = details?.query_sql ?? "";
        queryImpact.current = details?.stats
          ? getPerformanceImpactDescription(details.stats)
          : undefined;
        setShowShowQueryModal(true);
        break;
      case ActivityType.RanScript:
        scriptExecutionId.current = details?.script_execution_id ?? "";
        setShowScriptDetailsModal(true);
        break;
      case ActivityType.InstalledSoftware:
        setPackageInstallDetails({ ...details });
        break;
      case ActivityType.UninstalledSoftware:
        setPackageUninstallDetails({ ...details });
        break;
      case ActivityType.InstalledAppStoreApp:
        setAppInstallDetails({ ...details });
        break;
      case ActivityType.EnabledActivityAutomations:
      case ActivityType.EditedActivityAutomations:
        setActivityAutomationDetails({ ...details });
        break;
      case ActivityType.AddedSoftware:
      case ActivityType.EditedSoftware:
      case ActivityType.DeletedSoftware:
        setSoftwareDetails({ ...details });
        break;
      case ActivityType.AddedAppStoreApp:
      case ActivityType.EditedAppStoreApp:
      case ActivityType.DeletedAppStoreApp:
        setVppDetails({ ...details });
        break;
      case ActivityType.RanScriptBatch:
        setScriptBatchExecutionDetails({ ...details, created_at });
        break;
      default:
        break;
    }
  };

  const renderError = () => {
    return <DataError verticalPaddingSize="pad-large" />;
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
              <GlobalActivityItem
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
          <Pagination
            disablePrev={isFetchingActivities || !meta?.has_previous_results}
            disableNext={isFetchingActivities || !meta?.has_next_results}
            hidePagination={
              !isFetchingActivities &&
              !meta?.has_previous_results &&
              !meta?.has_next_results
            }
            onPrevPage={onLoadPrevious}
            onNextPage={onLoadNext}
          />
        )}
      {showShowQueryModal && (
        <ShowQueryModal
          query={queryShown.current}
          impact={queryImpact.current}
          onCancel={() => setShowShowQueryModal(false)}
        />
      )}
      {showScriptDetailsModal && (
        <RunScriptDetailsModal
          scriptExecutionId={scriptExecutionId.current}
          onCancel={() => setShowScriptDetailsModal(false)}
        />
      )}
      {packageInstallDetails && (
        <SoftwareInstallDetailsModal
          details={packageInstallDetails}
          onCancel={() => setPackageInstallDetails(null)}
        />
      )}
      {packageUninstallDetails && (
        <SoftwareUninstallDetailsModal
          details={packageUninstallDetails}
          onCancel={() => setPackageUninstallDetails(null)}
        />
      )}
      {appInstallDetails && (
        <AppInstallDetailsModal
          details={appInstallDetails}
          onCancel={() => setAppInstallDetails(null)}
        />
      )}
      {activityAutomationDetails && (
        <ActivityAutomationDetailsModal
          details={activityAutomationDetails}
          onCancel={() => setActivityAutomationDetails(null)}
        />
      )}
      {softwareDetails && (
        <SoftwareDetailsModal
          details={softwareDetails}
          onCancel={() => setSoftwareDetails(null)}
        />
      )}
      {vppDetails && (
        <VppDetailsModal
          details={vppDetails}
          onCancel={() => setVppDetails(null)}
        />
      )}
      {scriptBatchExecutionDetails && (
        <ScriptBatchSummaryModal
          scriptBatchExecutionDetails={scriptBatchExecutionDetails}
          onCancel={() => setScriptBatchExecutionDetails(null)}
          router={router}
        />
      )}
    </div>
  );
};

export default ActivityFeed;

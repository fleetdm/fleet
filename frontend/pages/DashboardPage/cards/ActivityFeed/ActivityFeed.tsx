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

import { AppInstallDetailsModal } from "components/ActivityDetails/InstallDetails/AppInstallDetails";
import { SoftwareInstallDetailsModal } from "components/ActivityDetails/InstallDetails/SoftwareInstallDetails/SoftwareInstallDetails";
import SoftwareUninstallDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";

import ActivityItem from "./ActivityItem";
import ActivityAutomationDetailsModal from "./components/ActivityAutomationDetailsModal";
import RunScriptDetailsModal from "./components/RunScriptDetailsModal/RunScriptDetailsModal";
import SoftwareDetailsModal from "./components/SoftwareDetailsModal";
import createMockActivity from "__mocks__/activityMock";

const baseClass = "activity-feed";
interface IActvityCardProps {
  setShowActivityFeedTitle: (showActivityFeedTitle: boolean) => void;
  setRefetchActivities: (refetch: () => void) => void;
  isPremiumTier: boolean;
}

const DEFAULT_PAGE_SIZE = 8;

const ActivityFeed = ({
  setShowActivityFeedTitle,
  setRefetchActivities,
  isPremiumTier,
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

  const handleDetailsClick = (
    activityType: ActivityType,
    details: IActivityDetails
  ) => {
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
        setSoftwareDetails({ ...details });
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

  // const activities = activitiesData?.activities;
  const activities = [
    createMockActivity({
      type: ActivityType.AddedSoftware,
      details: {
        software_title: "Software Title",
        software_package: "Software Package",
        self_service: true,
        labels_include_any: [
          { id: 1, name: "Label 1" },
          { id: 2, name: "Label 2" },
          { id: 3, name: "Label 3" },
        ],
      },
    }),
  ];
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
    </div>
  );
};

export default ActivityFeed;

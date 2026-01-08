import React, { useMemo, useRef, useState } from "react";
import { useQuery } from "react-query";
import { isEmpty } from "lodash";
import { InjectedRouter } from "react-router";

import paths from "router/paths";

import activitiesAPI, {
  IActivitiesResponse,
} from "services/entities/activities";

import { isAndroid } from "interfaces/platform";
import {
  resolveUninstallStatus,
  SoftwareInstallUninstallStatus,
  SCRIPT_PACKAGE_SOURCES,
} from "interfaces/software";
import { ActivityType, IActivityDetails } from "interfaces/activity";
import { PerformanceImpactIndicator } from "interfaces/schedulable_query";

import { getPerformanceImpactDescription } from "utilities/helpers";

import ShowQueryModal from "components/modals/ShowQueryModal";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import Pagination from "components/Pagination";
import EmptyTable from "components/EmptyTable";

import VppInstallDetailsModal from "components/ActivityDetails/InstallDetails/VppInstallDetailsModal";
import { SoftwareInstallDetailsModal } from "components/ActivityDetails/InstallDetails/SoftwareInstallDetailsModal/SoftwareInstallDetailsModal";
import SoftwareScriptDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareScriptDetailsModal/SoftwareScriptDetailsModal";
import SoftwareIpaInstallDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareIpaInstallDetailsModal";
import SoftwareUninstallDetailsModal, {
  ISWUninstallDetailsParentState,
} from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";
import { IShowActivityDetailsData } from "components/ActivityItem/ActivityItem";

import GlobalActivityItem from "./GlobalActivityItem";
import ActivityAutomationDetailsModal from "./components/ActivityAutomationDetailsModal";
import RunScriptDetailsModal from "./components/RunScriptDetailsModal/RunScriptDetailsModal";
import SoftwareDetailsModal from "./components/LibrarySoftwareDetailsModal";
import AppStoreDetailsModal from "./components/AppStoreDetailsModal/AppStoreDetailsModal";
import ActivityFeedFilters from "./components/ActivityFeedFilters";

const baseClass = "activity-feed";
interface IActvityCardProps {
  setShowActivityFeedTitle: (showActivityFeedTitle: boolean) => void;
  setRefetchActivities: (refetch: () => void) => void;
  isPremiumTier: boolean;
  router: InjectedRouter;
}

const DEFAULT_PAGE_SIZE = 8;

const generateDateFilter = (dateFilter: string) => {
  const startDate = new Date();
  const endDate = new Date();

  switch (dateFilter) {
    case "all":
      return {
        startDate: "",
        endDate: "",
      };
    case "today":
      startDate.setHours(0, 0, 0, 0);
      endDate.setHours(23, 59, 59, 999);
      break;
    case "yesterday":
      startDate.setDate(startDate.getDate() - 1);
      startDate.setHours(0, 0, 0, 0);
      endDate.setDate(endDate.getDate() - 1);
      endDate.setHours(23, 59, 59, 999);
      break;
    case "7d":
      startDate.setDate(startDate.getDate() - 7);
      break;
    case "30d":
      startDate.setDate(startDate.getDate() - 30);
      break;
    case "3m":
      startDate.setMonth(startDate.getMonth() - 3);
      break;
    case "12m":
      startDate.setMonth(startDate.getMonth() - 12);
      break;
    default:
      break;
  }

  return {
    startDate: startDate.toISOString(),
    endDate: endDate.toISOString(),
  }; // We convert to seconds since epoch as that is what the backend expects
};

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
  ] = useState<IActivityDetails | null>(null); // Also includes Android Play Store installs
  const [
    scriptPackageDetails,
    setScriptPackageDetails,
  ] = useState<IActivityDetails | null>(null);
  const [
    ipaPackageInstallDetails,
    setIpaPackageInstallDetails,
  ] = useState<IActivityDetails | null>(null);
  const [
    packageUninstallDetails,
    setPackageUninstallDetails,
  ] = useState<ISWUninstallDetailsParentState | null>(null);
  const [
    vppInstallDetails,
    setVppInstallDetails,
  ] = useState<IActivityDetails | null>(null);
  const [
    activityAutomationDetails,
    setActivityAutomationDetails,
  ] = useState<IActivityDetails | null>(null);
  const [
    softwareDetails,
    setSoftwareDetails,
  ] = useState<IActivityDetails | null>(null);
  const [
    appStoreDetails,
    setAppStoreDetails,
  ] = useState<IActivityDetails | null>(null);

  const [searchQuery, setSearchQuery] = useState("");
  const [createdAtDirection, setCreatedAtDirection] = useState("desc");
  const [dateFilter, setDateFilter] = useState("all");
  const [typeFilter, setTypeFilter] = useState<string[]>([""]);

  const queryShown = useRef("");
  const queryImpact = useRef<PerformanceImpactIndicator | undefined>(undefined);
  const scriptExecutionId = useRef("");

  const { startDate, endDate } = useMemo(() => generateDateFilter(dateFilter), [
    dateFilter,
  ]);

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
      query?: string;
      orderDirection?: string;
      startDate?: string;
      endDate?: string;
      typeFilter?: string[];
    }>
  >(
    [
      {
        scope: "activities",
        pageIndex,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDirection: createdAtDirection,
        startDate,
        endDate,
        typeFilter,
      },
    ],
    ({
      queryKey: [
        {
          pageIndex: page,
          perPage,
          query,
          orderDirection,
          startDate: queryStartDate,
          endDate: queryEndDate,
          typeFilter: queryTypeFilter,
        },
      ],
    }) => {
      return activitiesAPI.loadNext(
        page,
        perPage,
        query,
        orderDirection,
        queryStartDate,
        queryEndDate,
        queryTypeFilter
      );
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

  const handleDetailsClick = ({ type, details }: IShowActivityDetailsData) => {
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
        if (SCRIPT_PACKAGE_SOURCES.includes(details?.source || "")) {
          setScriptPackageDetails({ ...details });
        } else {
          details?.command_uuid
            ? setIpaPackageInstallDetails({ ...details })
            : setPackageInstallDetails({ ...details });
        }
        break;
      case ActivityType.UninstalledSoftware:
        setPackageUninstallDetails({
          ...details,
          softwareName:
            details?.software_display_name || details?.software_title || "",
          uninstallStatus: resolveUninstallStatus(details?.status),
          scriptExecutionId: details?.script_execution_id || "",
          hostDisplayName: details?.host_display_name,
        });
        break;
      case ActivityType.InstalledAppStoreApp:
        setVppInstallDetails({ ...details }); // Apple VPP + Android installs
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
        setAppStoreDetails({ ...details });
        break;
      case ActivityType.RanScriptBatch:
      case ActivityType.CanceledScriptBatch:
        router.push(
          paths.CONTROLS_SCRIPTS_BATCH_DETAILS(
            details?.batch_execution_id || ""
          )
        );
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
      <EmptyTable
        header="No activities match the current criteria"
        info="Try editing a query, updating your policies, or running a live query."
      />
    );
  };

  // Renders opaque information as activity feed is loading
  const opacity = isFetchingActivities ? { opacity: 0.4 } : { opacity: 1 };

  const activities = activitiesData?.activities;
  const meta = activitiesData?.meta;

  return (
    <div className={baseClass}>
      <ActivityFeedFilters
        searchQuery={searchQuery}
        typeFilter={typeFilter}
        dateFilter={dateFilter}
        createdAtDirection={createdAtDirection}
        setSearchQuery={setSearchQuery}
        setTypeFilter={setTypeFilter}
        setDateFilter={setDateFilter}
        setCreatedAtDirection={setCreatedAtDirection}
        setPageIndex={setPageIndex}
      />
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
      {scriptPackageDetails && (
        <SoftwareScriptDetailsModal
          details={scriptPackageDetails}
          onCancel={() => setScriptPackageDetails(null)}
        />
      )}
      {ipaPackageInstallDetails && (
        <SoftwareIpaInstallDetailsModal
          details={{
            appName:
              ipaPackageInstallDetails.software_display_name ||
              ipaPackageInstallDetails.software_title ||
              "",
            fleetInstallStatus: (ipaPackageInstallDetails.status ||
              "pending_install") as SoftwareInstallUninstallStatus,
            hostDisplayName: ipaPackageInstallDetails.host_display_name || "",
            commandUuid: ipaPackageInstallDetails.command_uuid || "",
          }}
          onCancel={() => setIpaPackageInstallDetails(null)}
        />
      )}
      {packageUninstallDetails && (
        <SoftwareUninstallDetailsModal
          {...packageUninstallDetails}
          hostDisplayName={packageUninstallDetails.hostDisplayName || ""}
          onCancel={() => setPackageUninstallDetails(null)}
        />
      )}
      {vppInstallDetails && (
        <VppInstallDetailsModal
          details={{
            appName:
              vppInstallDetails.software_display_name ||
              vppInstallDetails.software_title ||
              "",
            fleetInstallStatus: (vppInstallDetails.status ||
              "pending_install") as SoftwareInstallUninstallStatus,
            hostDisplayName: vppInstallDetails.host_display_name || "",
            commandUuid: vppInstallDetails.command_uuid || "",
            platform: vppInstallDetails.host_platform,
          }}
          onCancel={() => setVppInstallDetails(null)}
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
      {appStoreDetails && (
        <AppStoreDetailsModal
          details={appStoreDetails}
          onCancel={() => setAppStoreDetails(null)}
        />
      )}
    </div>
  );
};

export default ActivityFeed;

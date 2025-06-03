import React, { useMemo, useRef, useState } from "react";
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
import { SoftwareInstallDetailsModal } from "components/ActivityDetails/InstallDetails/SoftwareInstallDetails/SoftwareInstallDetails";
import SoftwareUninstallDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";
import { IShowActivityDetailsData } from "components/ActivityItem/ActivityItem";
import SearchField from "components/forms/fields/SearchField";
import CustomLink from "components/CustomLink";
import ActionsDropdown from "components/ActionsDropdown";

import GlobalActivityItem from "./GlobalActivityItem";
import ActivityAutomationDetailsModal from "./components/ActivityAutomationDetailsModal";
import RunScriptDetailsModal from "./components/RunScriptDetailsModal/RunScriptDetailsModal";
import SoftwareDetailsModal from "./components/SoftwareDetailsModal";
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

const SORT_OPTIONS = [
  { label: "Newest", value: "desc" },
  { label: "Oldest", value: "asc" },
];

const TYPE_FILTER_OPTIONS: { label: string; value: string }[] = Object.values(
  ActivityType
)
  .map((type) => ({
    label: type.replace(/_/gi, " ").toLowerCase(),
    value: type,
  }))
  .sort((a, b) => a.label.localeCompare(b.label));

TYPE_FILTER_OPTIONS.unshift({
  label: "all types",
  value: "",
});

const DATE_FILTER_OPTIONS = [
  { label: "All time", value: "all" },
  { label: "Today", value: "today" },
  { label: "Yesterday", value: "yesterday" },
  { label: "Last 7 days", value: "7d" },
  { label: "Last 30 days", value: "30d" },
  { label: "Last 3 months", value: "3m" },
  { label: "Last 12 months", value: "12m" },
];

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
  };
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

  const [searchQuery, setSearchQuery] = useState("");
  const [createdAtDirection, setCreatedAtDirection] = useState("desc");
  const [dateFilter, setDateFilter] = useState("all");
  const [typeFilter, setTypeFilter] = useState<string[]>([]);

  const queryShown = useRef("");
  const queryImpact = useRef<string | undefined>(undefined);
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

  console.log(typeFilter);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__search-filter`}>
        <SearchField
          placeholder="Search activities by user's name or email..."
          defaultValue={searchQuery}
          onChange={(value) => {
            setSearchQuery(value);
            setPageIndex(0);
          }}
          icon="search"
        />
        <div className={`${baseClass}__dropdown-filters`}>
          <div className={`${baseClass}__filters`}>
            <ActionsDropdown
              className={`${baseClass}__type-filter-dropdown`}
              options={TYPE_FILTER_OPTIONS}
              placeholder={`Type: ${
                typeFilter?.[0]?.replace(/_/g, " ") || "All"
              }`}
              onChange={(value: string) => {
                setTypeFilter((prev) => {
                  // TODO: multiple selections
                  return [value];
                });
                setPageIndex(0); // Reset to first page on sort change
              }}
            />
            <ActionsDropdown
              className={`${baseClass}__date-filter-dropdown`}
              options={DATE_FILTER_OPTIONS}
              placeholder={`Date: ${
                DATE_FILTER_OPTIONS.find(
                  (option) => option.value === dateFilter
                )?.label
              }`}
              onChange={(value: string) => {
                if (value === createdAtDirection) {
                  return; // No change in sort direction
                }
                setDateFilter(value);
                setPageIndex(0); // Reset to first page on sort change
              }}
            />
          </div>
          <ActionsDropdown
            className={`${baseClass}__sort-created-at-dropdown`}
            options={SORT_OPTIONS}
            placeholder={`Sort by: ${
              createdAtDirection === "asc" ? "Oldest" : "Newest"
            }`}
            onChange={(value: string) => {
              if (value === createdAtDirection) {
                return; // No change in sort direction
              }
              setCreatedAtDirection(value);
              setPageIndex(0); // Reset to first page on sort change
            }}
          />
        </div>
      </div>
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

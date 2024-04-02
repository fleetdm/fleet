import React, { useEffect, useMemo } from "react";
import { useQuery } from "react-query";

import {
  OS_END_OF_LIFE_LINK_BY_PLATFORM,
  OS_VENDOR_BY_PLATFORM,
} from "interfaces/operating_system";
import { SelectedPlatform } from "interfaces/platform";
import {
  getOSVersions,
  IGetOSVersionsQueryKey,
  IOSVersionsResponse,
  OS_VERSIONS_API_SUPPORTED_PLATFORMS,
} from "services/entities/operating_systems";
import { PLATFORM_DISPLAY_NAMES } from "utilities/constants";

import TableContainer from "components/TableContainer";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import LastUpdatedText from "components/LastUpdatedText";
import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
import { AxiosError } from "axios";

import generateTableHeaders from "./OperatingSystemsTableConfig";

interface IOperatingSystemsCardProps {
  currentTeamId: number | undefined;
  selectedPlatform: SelectedPlatform;
  showTitle: boolean;
  /** controls the displaying of description text under the title. Defaults to `true` */
  showDescription?: boolean;
  /** controls the displaying of the **Name** column in the table. Defaults to `true` */
  includeNameColumn?: boolean;
  setShowTitle: (showTitle: boolean) => void;
  setTitleDetail?: (content: JSX.Element | string | null) => void;
  setTitleDescription?: (content: JSX.Element | string | null) => void;
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 8;
const baseClass = "operating-systems";

const EmptyOperatingSystems = (platform: SelectedPlatform): JSX.Element => (
  <EmptyTable
    className={`${baseClass}__os-empty-table`}
    header={`No${
      ` ${PLATFORM_DISPLAY_NAMES[platform]}` || ""
    } operating systems detected`}
    info="This report is updated every hour to protect the performance of your
      devices."
  />
);

const OperatingSystems = ({
  currentTeamId,
  selectedPlatform,
  showTitle,
  showDescription = true,
  includeNameColumn = true,
  setShowTitle,
  setTitleDetail,
  setTitleDescription,
}: IOperatingSystemsCardProps): JSX.Element => {
  const { data: osInfo, error, isFetching } = useQuery<
    IOSVersionsResponse,
    AxiosError,
    IOSVersionsResponse,
    IGetOSVersionsQueryKey[]
  >(
    [
      {
        scope: "os_versions",
        platform: selectedPlatform !== "all" ? selectedPlatform : undefined,
        teamId: currentTeamId,
      },
    ],
    ({ queryKey: [{ platform, teamId }] }) => {
      return getOSVersions({
        platform,
        teamId,
      });
    },
    {
      enabled: OS_VERSIONS_API_SUPPORTED_PLATFORMS.includes(selectedPlatform),
      staleTime: 10000,
      keepPreviousData: true,
      retry: 0,
    }
  );

  const renderDescription = () => {
    if (selectedPlatform === "chrome") {
      return (
        <p>
          Chromebooks automatically receive updates from Google until their
          auto-update expiration date.{" "}
          <CustomLink
            url="https://fleetdm.com/learn-more-about/chromeos-updates"
            text="See supported devices"
            newTab
            multiline
          />
        </p>
      );
    }
    if (
      showDescription &&
      OS_VENDOR_BY_PLATFORM[selectedPlatform] &&
      OS_END_OF_LIFE_LINK_BY_PLATFORM[selectedPlatform]
    )
      return (
        <p>
          {OS_VENDOR_BY_PLATFORM[selectedPlatform]} releases updates and fixes
          for supported operating systems.{" "}
          <CustomLink
            url={OS_END_OF_LIFE_LINK_BY_PLATFORM[selectedPlatform]}
            text="See supported operating systems"
            newTab
            multiline
          />
        </p>
      );
    return null;
  };
  const titleDetail = osInfo?.counts_updated_at ? (
    <LastUpdatedText
      lastUpdatedAt={osInfo?.counts_updated_at}
      whatToRetrieve="operating systems"
    />
  ) : null;

  useEffect(() => {
    if (isFetching) {
      setShowTitle(false);
      setTitleDescription?.(null);
      setTitleDetail?.(null);
      return;
    }
    setShowTitle(true);
    if (osInfo?.os_versions?.length) {
      setTitleDescription?.(renderDescription());
      setTitleDetail?.(titleDetail);
      return;
    }
    setTitleDescription?.(null);
    setTitleDetail?.(null);
  }, [isFetching, osInfo, setTitleDescription, setTitleDetail]);

  const tableHeaders = useMemo(
    () => generateTableHeaders(currentTeamId, undefined, { includeName: true }),
    [includeNameColumn, currentTeamId]
  );

  const showPaginationControls = (osInfo?.os_versions?.length || 0) > 8;

  // Renders opaque information as host information is loading
  const opacity = isFetching || !showTitle ? { opacity: 0 } : { opacity: 1 };

  return (
    <div className={baseClass}>
      {isFetching && (
        <div className="spinner">
          <Spinner />
        </div>
      )}
      <div style={opacity}>
        {error?.status && error?.status >= 500 ? (
          <TableDataError card />
        ) : (
          <TableContainer
            columnConfigs={tableHeaders}
            data={osInfo?.os_versions || []}
            isLoading={isFetching}
            defaultSortHeader={DEFAULT_SORT_HEADER}
            defaultSortDirection={DEFAULT_SORT_DIRECTION}
            resultsTitle="Operating systems"
            emptyComponent={() => EmptyOperatingSystems(selectedPlatform)}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableCount
            isClientSidePagination={showPaginationControls}
            disablePagination={!showPaginationControls}
            pageSize={PAGE_SIZE}
          />
        )}
      </div>
    </div>
  );
};

export default OperatingSystems;

import React, { useEffect } from "react";
import { useQuery } from "react-query";

import {
  OS_END_OF_LIFE_LINK_BY_PLATFORM,
  OS_VENDOR_BY_PLATFORM,
} from "interfaces/operating_system";
import { ISelectedPlatform } from "interfaces/platform";
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

import ExternalLinkIcon from "../../../../../assets/images/icon-external-link-12x12@2x.png";

import generateTableHeaders from "./OperatingSystemsTableConfig";

interface IOperatingSystemsCardProps {
  currentTeamId: number | undefined;
  selectedPlatform: ISelectedPlatform;
  showTitle: boolean;
  setShowTitle: (showTitle: boolean) => void;
  setTitleDetail?: (content: JSX.Element | string | null) => void;
  setTitleDescription?: (content: JSX.Element | string | null) => void;
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 8;
const baseClass = "operating-systems";

const EmptyOperatingSystems = (platform: ISelectedPlatform): JSX.Element => (
  <div className={`${baseClass}__empty-os`}>
    <h1>{`No${
      ` ${PLATFORM_DISPLAY_NAMES[platform]}` || ""
    } operating systems detected.`}</h1>
    <p>
      {`Did you add ${`${PLATFORM_DISPLAY_NAMES[platform]} ` || ""}hosts to
      Fleet? Try again in about an hour as the system catches up.`}
    </p>
  </div>
);

const OperatingSystems = ({
  currentTeamId,
  selectedPlatform,
  showTitle,
  setShowTitle,
  setTitleDetail,
  setTitleDescription,
}: IOperatingSystemsCardProps): JSX.Element => {
  const { data: osInfo, error, isFetching } = useQuery<
    IOSVersionsResponse,
    Error,
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
    }
  );

  const description =
    OS_VENDOR_BY_PLATFORM[selectedPlatform] &&
    OS_END_OF_LIFE_LINK_BY_PLATFORM[selectedPlatform] ? (
      <p>
        {OS_VENDOR_BY_PLATFORM[selectedPlatform]} releases updates and fixes for
        supported operating systems.{" "}
        <a
          href={OS_END_OF_LIFE_LINK_BY_PLATFORM[selectedPlatform]}
          target="_blank"
          rel="noreferrer noopener"
        >
          See supported operating{" "}
          <span className="no-wrap">
            systems
            <img alt="Open external link" src={ExternalLinkIcon} />
          </span>
        </a>
      </p>
    ) : null;

  const titleDetail = osInfo?.counts_updated_at ? (
    <LastUpdatedText
      lastUpdatedAt={osInfo?.counts_updated_at}
      whatToRetrieve={"operating systems"}
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
      setTitleDescription?.(description);
      setTitleDetail?.(titleDetail);
      return;
    }
    setTitleDescription?.(null);
    setTitleDetail?.(null);
  }, [isFetching, osInfo, setTitleDescription, setTitleDetail]);

  const tableHeaders = generateTableHeaders();
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
        {error ? (
          <TableDataError card />
        ) : (
          <TableContainer
            columns={tableHeaders}
            data={osInfo?.os_versions || []}
            isLoading={isFetching}
            defaultSortHeader={DEFAULT_SORT_HEADER}
            defaultSortDirection={DEFAULT_SORT_DIRECTION}
            hideActionButton
            resultsTitle={"Operating systems"}
            emptyComponent={() => EmptyOperatingSystems(selectedPlatform)}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableCount
            disableActionButton
            isClientSidePagination={showPaginationControls}
            disablePagination={!showPaginationControls}
            pageSize={PAGE_SIZE}
            highlightOnHover
          />
        )}
      </div>
    </div>
  );
};

export default OperatingSystems;

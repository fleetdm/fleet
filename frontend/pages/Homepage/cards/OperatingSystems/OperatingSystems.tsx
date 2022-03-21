import React from "react";
import { useQuery } from "react-query";

import operatingSystemsAPI, {
  IOperatingSystemsResponse,
} from "services/entities/operating_systems";

import TableContainer from "components/TableContainer";
// @ts-ignore
import Spinner from "components/Spinner";
import renderLastUpdatedText from "components/LastUpdatedText";

import generateTableHeaders from "./OperatingSystemsTableConfig";

interface IOperatingSystemsCardProps {
  currentTeamId: number | undefined;
  selectedPlatform: string;
  showOperatingSystemsUI: boolean;
  setShowOperatingSystemsUI: (showOperatingSystemsTitle: boolean) => void;
  setTitleDetail?: (content: JSX.Element | string | null) => void;
}

type IOsQueryPlatform = "darwin" | "linux" | "windows"; // TODO: replace these with imports once my platform PR is merged
const OS_API_SUPPORTED_PLATFORMS: IOsQueryPlatform[] = ["darwin"];

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 8;
const baseClass = "operating-systems";

const EmptyOperatingSystems = (): JSX.Element => (
  <div className={`${baseClass}__empty-munki`}>
    <h1>Unable to detect operating systems versions.</h1>
    <p>
      To see operating systems versions, deploy&nbsp;
      <a
        href="https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer"
        target="_blank"
        rel="noopener noreferrer"
      >
        Fleet&apos;s osquery installer
      </a>
      .
    </p>
  </div>
);

const OperatingSystems = ({
  currentTeamId,
  selectedPlatform,
  showOperatingSystemsUI,
  setShowOperatingSystemsUI,
  setTitleDetail,
}: IOperatingSystemsCardProps): JSX.Element => {
  const platform = selectedPlatform as IOsQueryPlatform;

  const { data: osInfo, isFetching } = useQuery<
    IOperatingSystemsResponse,
    Error,
    IOperatingSystemsResponse,
    Array<{
      scope: string;
      platform: IOsQueryPlatform;
      teamId: number | undefined;
    }>
  >(
    [{ scope: "os_version", platform, teamId: currentTeamId }],
    ({ queryKey: [{ platform, teamId }] }) => {
      return operatingSystemsAPI.getVersions({
        platform,
        teamId: currentTeamId,
      });
    },
    {
      enabled: OS_API_SUPPORTED_PLATFORMS.includes(platform),
      keepPreviousData: true,
      onSuccess: (data) => {
        setShowOperatingSystemsUI(true);
        setTitleDetail &&
          setTitleDetail(
            renderLastUpdatedText(data.counts_updated_at, "operating systems")
          );
      },
    }
  );

  const tableHeaders = generateTableHeaders();

  // Renders opaque information as host information is loading
  const opacity = showOperatingSystemsUI ? { opacity: 1 } : { opacity: 0 };

  return (
    <div className={baseClass}>
      {!showOperatingSystemsUI && (
        <div className="spinner">
          <Spinner />
        </div>
      )}
      <div style={opacity}>
        <TableContainer
          columns={tableHeaders}
          data={osInfo?.os_versions || []}
          isLoading={isFetching}
          defaultSortHeader={DEFAULT_SORT_HEADER}
          defaultSortDirection={DEFAULT_SORT_DIRECTION}
          hideActionButton
          resultsTitle={"Operating systems"}
          emptyComponent={EmptyOperatingSystems}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disableCount
          disableActionButton
          disablePagination
          pageSize={PAGE_SIZE}
        />
      </div>
    </div>
  );
};

export default OperatingSystems;

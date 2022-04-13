import React from "react";
import { useQuery } from "react-query";

import { IOsqueryPlatform } from "interfaces/platform";
import operatingSystemsAPI, {
  IOperatingSystemsResponse,
} from "services/entities/operating_systems";
import { PLATFORM_DISPLAY_NAMES } from "utilities/constants";

import TableContainer from "components/TableContainer";
import Spinner from "components/Spinner";
import renderLastUpdatedText from "components/LastUpdatedText";

import generateTableHeaders from "./OperatingSystemsTableConfig";

interface IOperatingSystemsCardProps {
  currentTeamId: number | undefined;
  selectedPlatform: IOsqueryPlatform;
  showOperatingSystemsUI: boolean;
  setShowOperatingSystemsUI: (showOperatingSystemsTitle: boolean) => void;
  setTitleDetail?: (content: JSX.Element | string | null) => void;
}

// TODO: add platforms to this constant as new ones are supported
const OS_API_SUPPORTED_PLATFORMS: IOsqueryPlatform[] = ["darwin"];

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 8;
const baseClass = "operating-systems";

const EmptyOperatingSystems = (platform: IOsqueryPlatform): JSX.Element => (
  <div className={`${baseClass}__empty-os`}>
    <h1>{`No ${
      PLATFORM_DISPLAY_NAMES[platform] || "supported"
    } operating systems detected`}</h1>
    <p>
      {`Did you add ${`${PLATFORM_DISPLAY_NAMES[platform]} ` || ""}hosts to
      Fleet? Try again in a few seconds as the system catches up.`}
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
  const { data: osInfo, error, isFetching } = useQuery<
    IOperatingSystemsResponse,
    Error,
    IOperatingSystemsResponse,
    Array<{
      scope: string;
      platform: IOsqueryPlatform;
      teamId: number | undefined;
    }>
  >(
    [
      {
        scope: "os_version",
        platform: selectedPlatform,
        teamId: currentTeamId,
      },
    ],
    ({ queryKey: [{ platform, teamId }] }) => {
      return operatingSystemsAPI.getVersions({
        platform,
        teamId,
      });
    },
    {
      enabled: OS_API_SUPPORTED_PLATFORMS.includes(selectedPlatform),
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
  console.log("teamId: ", currentTeamId);

  // TODO: Error states? Product says if any card on homepage fails the whole page should 500. Is
  // that really what we want to happen? Do we want that to happen always? What if just one card
  // fails? What if platform or team filter is applied?
  // Currenly none of the homepage cards behave this way AFAICT.
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
          emptyComponent={() => EmptyOperatingSystems(selectedPlatform)}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disableCount
          disableActionButton
          isClientSidePagination
          pageSize={PAGE_SIZE}
        />
      </div>
    </div>
  );
};

export default OperatingSystems;

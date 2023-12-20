import React, { useContext, useMemo } from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";

import softwareAPI, {
  ISoftwareVersionResponse,
} from "services/entities/software";
import hostsCountAPI, {
  IHostsCountQueryKey,
  IHostsCountResponse,
} from "services/entities/host_count";
import { ISoftwareVersion, formatSoftwareType } from "interfaces/software";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";
import { AppContext } from "context/app";

import MainContent from "components/MainContent";
import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
import TableDataError from "components/DataError";
import Spinner from "components/Spinner";

import generateSoftwareVersionDetailsTableConfig from "./SoftwareVersionDetailsTableConfig";
import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";

const baseClass = "software-version-details-page";

interface ISoftwareVersionDetailsRouteParams {
  id: string;
}

type ISoftwareTitleDetailsPageProps = RouteComponentProps<
  undefined,
  ISoftwareVersionDetailsRouteParams
>;

const NoVulnsDetected = (): JSX.Element => {
  return (
    <EmptyTable
      header="No vulnerabilities detected for this software item."
      info={
        <>
          Expecting to see vulnerabilities?{" "}
          <CustomLink
            url={GITHUB_NEW_ISSUE_LINK}
            text="File an issue on GitHub"
            newTab
          />
        </>
      }
    />
  );
};

const SoftwareVersionDetailsPage = ({
  routeParams,
}: ISoftwareTitleDetailsPageProps) => {
  const versionId = parseInt(routeParams.id, 10);
  const { isPremiumTier, isSandboxMode } = useContext(AppContext);

  const {
    data: softwareVersion,
    isLoading: isSoftwareVersionLoading,
    isError: isSoftwareVersionError,
  } = useQuery<ISoftwareVersionResponse, Error, ISoftwareVersion>(
    ["software-version", versionId],
    () => softwareAPI.getSoftwareVersion(versionId),
    {
      select: (data) => data.software,
    }
  );

  // TODO: Confirm desired UX for error and loading states
  const {
    data: hostsCount,
    // isError: isHostsCountError,
    // isLoading: isHostsCountLoading,
  } = useQuery<IHostsCountResponse, Error, number, IHostsCountQueryKey[]>(
    [{ scope: "hosts_count", softwareVersionId: versionId }],
    ({ queryKey }) => hostsCountAPI.load(queryKey[0]),
    {
      keepPreviousData: true,
      staleTime: 10000, // stale time can be adjusted if fresher data is desired
      select: (data) => data.count,
    }
  );

  const tableHeaders = useMemo(
    () =>
      generateSoftwareVersionDetailsTableConfig(
        Boolean(isPremiumTier),
        Boolean(isSandboxMode)
      ),
    [isPremiumTier, isSandboxMode]
  );

  const renderContent = () => {
    if (isSoftwareVersionLoading) {
      return <Spinner />;
    }

    if (isSoftwareVersionError) {
      return <TableDataError className={`${baseClass}__table-error`} />;
    }

    if (!softwareVersion) {
      return null;
    }

    return (
      <>
        <SoftwareDetailsSummary
          id={softwareVersion.id}
          title={`${softwareVersion.name}, ${softwareVersion.version}`}
          type={formatSoftwareType(softwareVersion)}
          hosts={hostsCount ?? 0}
          queryParam="software_version_id"
          name={softwareVersion.name}
          source={softwareVersion.source}
        />
        <div className={`${baseClass}__vulnerabilities-section`}>
          <h2 className="section__header">Vulnerabilities</h2>
          {softwareVersion?.vulnerabilities?.length ? (
            <div className="vuln-table">
              <TableContainer
                columnConfigs={tableHeaders}
                data={softwareVersion.vulnerabilities}
                defaultSortHeader={isPremiumTier ? "epss_probability" : "cve"}
                defaultSortDirection={"desc"}
                emptyComponent={NoVulnsDetected}
                isAllPagesSelected={false}
                isLoading={isSoftwareVersionLoading}
                isClientSidePagination
                pageSize={20}
                resultsTitle={"vulnerabilities"}
                showMarkAllPages={false}
              />
            </div>
          ) : (
            <NoVulnsDetected />
          )}
        </div>
      </>
    );
  };

  return (
    <MainContent className={baseClass}>
      <>{renderContent()}</>
    </MainContent>
  );
};
export default SoftwareVersionDetailsPage;

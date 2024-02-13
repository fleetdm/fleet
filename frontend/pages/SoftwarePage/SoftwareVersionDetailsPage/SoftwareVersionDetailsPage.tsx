/** software/versions/:id */

import React from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import softwareAPI, {
  ISoftwareVersionResponse,
} from "services/entities/software";
import hostsCountAPI, {
  IHostsCountQueryKey,
  IHostsCountResponse,
} from "services/entities/host_count";
import { ISoftwareVersion, formatSoftwareType } from "interfaces/software";

import MainContent from "components/MainContent";
import TableDataError from "components/DataError";
import Spinner from "components/Spinner";

import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";
import SoftwareVulnerabilitiesTable from "../components/SoftwareVulnerabilitiesTable";

const baseClass = "software-version-details-page";

interface ISoftwareVersionDetailsRouteParams {
  id: string;
  team_id?: string;
}

interface ISoftwareOSDetailsPageProps {
  routeParams: ISoftwareVersionDetailsRouteParams;
  router: InjectedRouter;
}

const SoftwareVersionDetailsPage = ({
  routeParams,
  router,
}: ISoftwareOSDetailsPageProps) => {
  const versionId = parseInt(routeParams.id, 10);
  const teamId = routeParams.team_id
    ? parseInt(routeParams.team_id, 10)
    : undefined;

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

  const { data: hostsCount } = useQuery<
    IHostsCountResponse,
    Error,
    number,
    IHostsCountQueryKey[]
  >(
    [{ scope: "hosts_count", softwareVersionId: versionId }],
    ({ queryKey }) => hostsCountAPI.load(queryKey[0]),
    {
      keepPreviousData: true,
      staleTime: 10000, // stale time can be adjusted if fresher data is desired
      select: (data) => data.count,
    }
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
          title={`${softwareVersion.name}, ${softwareVersion.version}`}
          type={formatSoftwareType(softwareVersion)}
          hosts={hostsCount ?? 0}
          queryParams={{ software_version_id: softwareVersion.id }}
          name={softwareVersion.name}
          source={softwareVersion.source}
        />
        <div className={`${baseClass}__vulnerabilities-section`}>
          <h2 className="section__header">Vulnerabilities</h2>
          <SoftwareVulnerabilitiesTable
            data={softwareVersion.vulnerabilities ?? []}
            itemName="software item"
            isLoading={isSoftwareVersionLoading}
            router={router}
            teamId={teamId}
          />
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

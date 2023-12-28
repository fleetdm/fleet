import React from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";

import osVersionsAPI, {
  IOSVersionResponse,
} from "services/entities/operating_systems";
import { IOperatingSystemVersion } from "interfaces/operating_system";

import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";

import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";
import SoftwareVulnerabilitiesTable from "../components/SoftwareVulnerabilitiesTable";

const baseClass = "software-os-details-page";

interface ISoftwareOSDetailsRouteParams {
  id: string;
}

type ISoftwareOSDetailsPageProps = RouteComponentProps<
  undefined,
  ISoftwareOSDetailsRouteParams
>;

const SoftwareOSDetailsPage = ({
  routeParams,
}: ISoftwareOSDetailsPageProps) => {
  // TODO: handle non integer values
  const osVersionId = parseInt(routeParams.id, 10);

  const { data, isLoading, isError } = useQuery<
    IOSVersionResponse,
    Error,
    IOperatingSystemVersion
  >(
    ["osVersionById", osVersionId],
    () => osVersionsAPI.getOSVersion(osVersionId),
    {
      select: (res) => res.os_version,
    }
  );

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <TableDataError className={`${baseClass}__table-error`} />;
    }

    if (!data) {
      return null;
    }

    return (
      <>
        <SoftwareDetailsSummary
          id={data.id}
          title={data.name}
          hosts={data.hosts_count}
          queryParam="software_title_id"
          name={data.name}
        />
        {/* TODO: can we use Card here for card styles */}
        <div className={`${baseClass}__versions-section`}>
          <h2>Vulnerabilities</h2>
          <SoftwareVulnerabilitiesTable
            data={data.vulnerabilities}
            isLoading={isLoading}
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

export default SoftwareOSDetailsPage;

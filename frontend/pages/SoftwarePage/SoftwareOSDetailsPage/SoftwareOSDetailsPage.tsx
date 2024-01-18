import React from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";

import osVersionsAPI, {
  IOSVersionResponse,
} from "services/entities/operating_systems";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { SUPPORT_LINK } from "utilities/constants";

import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";
import SoftwareVulnerabilitiesTable from "../components/SoftwareVulnerabilitiesTable";

const baseClass = "software-os-details-page";

interface INotSupportedVulnProps {
  platform: string;
}

const NotSupportedVuln = ({ platform }: INotSupportedVulnProps) => {
  return (
    <EmptyTable
      header="Vulnerabilities are not supported for this type of host"
      info={
        <>
          Interested in vulnerability management for{" "}
          {platform === "chrome" ? "Chromebooks" : "Linux hosts"}?{" "}
          <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
        </>
      }
    />
  );
};

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

  const renderTabel = () => {
    if (!data) {
      return null;
    }

    if (data.platform !== "darwin" && data.platform !== "windows") {
      return <NotSupportedVuln platform={data.platform} />;
    }

    return (
      <SoftwareVulnerabilitiesTable
        data={data.vulnerabilities}
        itemName="version"
        isLoading={isLoading}
      />
    );
  };

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
          title={`${data.name} ${data.version}`}
          hosts={data.hosts_count}
          queryParams={{ os_name: data.name_only, os_version: data.version }}
          name={data.name}
        />
        {/* TODO: can we use Card here for card styles */}
        <div className={`${baseClass}__vulnerabilities-section`}>
          <h2>Vulnerabilities</h2>
          {renderTabel()}
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

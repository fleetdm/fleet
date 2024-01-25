import React from "react";
import { useQuery } from "react-query";

import osVersionsAPI, {
  IOSVersionsResponse,
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

interface ISoftwareOSDetailsPageProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  location: { query: { name: string; version: string } }; // no type in react-router v3
}

const SoftwareOSDetailsPage = ({ location }: ISoftwareOSDetailsPageProps) => {
  const name = location.query.name;
  const osVersion = location.query.version;
  const { data: osVersionDetails, isLoading, isError } = useQuery<
    IOSVersionsResponse,
    Error,
    IOperatingSystemVersion
  >(
    ["osVersionDetails", name, osVersion],
    () => osVersionsAPI.getOSVersion({ os_name: name, os_version: osVersion }),
    {
      select: (res) => res.os_versions[0],
    }
  );

  const renderTable = () => {
    if (!osVersionDetails) {
      return null;
    }

    if (
      osVersionDetails.platform !== "darwin" &&
      osVersionDetails.platform !== "windows"
    ) {
      return <NotSupportedVuln platform={osVersionDetails.platform} />;
    }

    return (
      <SoftwareVulnerabilitiesTable
        data={osVersionDetails.vulnerabilities}
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

    if (!osVersionDetails) {
      return null;
    }

    return (
      <>
        <SoftwareDetailsSummary
          title={`${osVersionDetails.name} ${osVersionDetails.version}`}
          hosts={osVersionDetails.hosts_count}
          queryParams={{
            os_name: osVersionDetails.name_only,
            os_version: osVersionDetails.version,
          }}
          name={osVersionDetails.name}
        />
        {/* TODO: can we use Card here for card styles */}
        <div className={`${baseClass}__vulnerabilities-section`}>
          <h2>Vulnerabilities</h2>
          {renderTable()}
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

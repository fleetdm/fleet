import React from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import {
  getOSVersions,
  IOSVersionsResponse,
} from "services/entities/operating_systems";

import LastUpdatedText from "components/LastUpdatedText";
import SectionHeader from "components/SectionHeader";

import OSVersionTable from "../OSVersionTable";
import DataError from "components/DataError";

const baseClass = "os-updates-current-version-section";

interface ICurrentVersionSectionProps {
  currentTeamId: number;
}

const CurrentVersionSection = ({
  currentTeamId,
}: ICurrentVersionSectionProps) => {
  const { data, isError, isLoading: isLoadingOsVersions } = useQuery<
    IOSVersionsResponse,
    AxiosError
  >(["os_versions", currentTeamId], () => getOSVersions(), {
    retry: false,
    refetchOnWindowFocus: false,
  });

  const generateSubTitleText = () => {
    return (
      <LastUpdatedText
        lastUpdatedAt={data?.counts_updated_at}
        whatToRetrieve={"operating systems"}
      />
    );
  };

  if (!data) {
    return null;
  }

  if (isError) {
    return <DataError />;
  }

  const renderTable = () => {
    if (isError) {
      return (
        <DataError
          description="Refresh the page to try again."
          excludeIssueLink
        />
      );
    }

    // We only want to show windows and mac versions atm.
    const filteredOSVersionData = data.os_versions.filter((osVersion) => {
      return (
        osVersion.platform === "windows" || osVersion.platform === "darwin"
      );
    });

    return (
      <OSVersionTable
        osVersionData={filteredOSVersionData}
        isLoading={isLoadingOsVersions}
      />
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader
        title="Current versions"
        subTitle={generateSubTitleText()}
      />
      {renderTable()}
    </div>
  );
};

export default CurrentVersionSection;

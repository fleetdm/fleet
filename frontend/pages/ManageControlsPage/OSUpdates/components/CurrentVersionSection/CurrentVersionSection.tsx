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

const baseClass = "os-updates-current-version-section";

interface ICurrentVersionSectionProps {
  currentTeamId: number;
}

const CurrentVersionSection = ({
  currentTeamId,
}: ICurrentVersionSectionProps) => {
  const { data, error, isLoading: isLoadingOsVersions } = useQuery<
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

  // We only want to show windows and mac versions atm.
  const filteredOSVersionData = data.os_versions.filter((osVersion) => {
    return osVersion.platform === "windows" || osVersion.platform === "darwin";
  });

  return (
    <div className={baseClass}>
      <SectionHeader
        title="Current versions"
        subTitle={generateSubTitleText()}
      />
      <OSVersionTable
        osVersionData={filteredOSVersionData}
        isLoading={isLoadingOsVersions}
      />
    </div>
  );
};

export default CurrentVersionSection;

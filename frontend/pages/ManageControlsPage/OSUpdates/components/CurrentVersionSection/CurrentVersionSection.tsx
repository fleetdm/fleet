import React from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import { IOperatingSystemVersion } from "interfaces/operating_system";
import {
  getOSVersions,
  IOSVersionsResponse,
} from "services/entities/operating_systems";

import LastUpdatedText from "components/LastUpdatedText";
import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";

import OSVersionTable from "../OSVersionTable";
import { OSUpdatesSupportedPlatform } from "../../OSUpdates";
import OSVersionsEmptyState from "../OSVersionsEmptyState";

/** This overrides the `platform` attribute on IOperatingSystemVersion so that only our filtered platforms (currently
 * "darwin" and "windows") values are included */
export type IFilteredOperatingSystemVersion = Omit<
  IOperatingSystemVersion,
  "platform"
> & {
  platform: OSUpdatesSupportedPlatform;
};

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
  >(
    ["os_versions", currentTeamId],
    () => getOSVersions({ teamId: currentTeamId }),
    {
      retry: false,
      refetchOnWindowFocus: false,
    }
  );

  const generateSubTitleText = () => {
    return (
      <LastUpdatedText
        lastUpdatedAt={data?.counts_updated_at}
        whatToRetrieve="operating systems"
      />
    );
  };

  if (!data) {
    return null;
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

    if (!data.os_versions) {
      return <OSVersionsEmptyState />;
    }

    // We only want to show windows and mac versions atm.
    const filteredOSVersionData = data.os_versions.filter((osVersion) => {
      return (
        osVersion.platform === "windows" || osVersion.platform === "darwin"
      );
    }) as IFilteredOperatingSystemVersion[];

    return (
      <OSVersionTable
        osVersionData={filteredOSVersionData}
        currentTeamId={currentTeamId}
        isLoading={isLoadingOsVersions}
      />
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader
        title="Current versions"
        subTitle={generateSubTitleText()}
        className={`${baseClass}__header`}
      />
      {renderTable()}
    </div>
  );
};

export default CurrentVersionSection;

import React, { useEffect } from "react";
import paths from "router/paths";

import { ILabelSummary } from "interfaces/label";
import { PLATFORM_NAME_TO_LABEL_NAME } from "utilities/constants";

import DataError from "components/DataError";
import SummaryTile from "./SummaryTile";

import SummaryTile from "../HostsSummary/SummaryTile";
import WindowsIcon from "../../../../../assets/images/icon-windows-48x48@2x.png";
import LinuxIcon from "../../../../../assets/images/icon-linux-48x48@2x.png";
import MacIcon from "../../../../../assets/images/icon-mac-48x48@2x.png";

const baseClass = "hosts-summary";

interface IHostSummaryProps {
  currentTeamId: number | undefined;
  macCount: number;
  windowsCount: number;
  linuxCount: number;
  isLoadingHostsSummary: boolean;
  showHostsUI: boolean;
  errorHosts: boolean;
  selectedPlatform: string;
  labels?: ILabelSummary[];
  setActionURL?: (url: string) => void;
}

const HostsSummary = ({
  currentTeamId,
  macCount,
  windowsCount,
  linuxCount,
  isLoadingHostsSummary,
  showHostsUI,
  errorHosts,
  selectedPlatform,
  labels,
  setActionURL,
}: IHostSummaryProps): JSX.Element => {
  const { MANAGE_HOSTS } = paths;

  const getLabel = (
    labelString: string,
    summaryLabels: ILabelSummary[]
  ): ILabelSummary | undefined => {
    return Object.values(summaryLabels).find((label: ILabelSummary) => {
      return label.label_type === "builtin" && label.name === labelString;
    });
  };

  // build the manage hosts URL
  useEffect(() => {
    if (labels) {
      let hostsURL = MANAGE_HOSTS;

      if (selectedPlatform) {
        const labelValue =
          PLATFORM_NAME_TO_LABEL_NAME[
            selectedPlatform as keyof typeof PLATFORM_NAME_TO_LABEL_NAME
          ];
        hostsURL += `/manage/labels/${getLabel(labelValue, labels)?.id}`;
      }

      if (currentTeamId) {
        hostsURL += `/?team_id=${currentTeamId}`;
      }

      setActionURL && setActionURL(hostsURL);
    }
  }, [labels, selectedPlatform, currentTeamId]);

  // Renders semi-transparent screen as host information is loading
  let opacity = { opacity: 0 };
  if (showHostsUI) {
    opacity = isLoadingHostsSummary ? { opacity: 0.4 } : { opacity: 1 };
  }

  const renderMacCount = () => (
    <SummaryTile
      icon={MacIcon}
      count={macCount}
      isLoading={isLoadingHostsSummary}
      showUI={showHostsUI}
      title="macOS hosts"
      path={paths.MANAGE_HOSTS_LABEL(7)}
    />
  );

  const renderWindowsCount = () => (
    <SummaryTile
      icon={WindowsIcon}
      count={windowsCount}
      isLoading={isLoadingHostsSummary}
      showUI={showHostsUI}
      title="Windows hosts"
      path={paths.MANAGE_HOSTS_LABEL(10)}
    />
  );

  const renderLinuxCount = () => (
    <SummaryTile
      icon={LinuxIcon}
      count={linuxCount}
      isLoading={isLoadingHostsSummary}
      showUI={showHostsUI}
      title="Linux hosts"
      path={paths.MANAGE_HOSTS_LABEL(12)}
    />
  );

  const renderCounts = () => {
    switch (selectedPlatform) {
      case "darwin":
        return renderMacCount();
      case "windows":
        return renderWindowsCount();
      case "linux":
        return renderLinuxCount();
      default:
        return (
          <>
            {renderMacCount()}
            {renderWindowsCount()}
            {renderLinuxCount()}
          </>
        );
    }
  };

  if (errorHosts && !isLoadingHostsSummary) {
    return <DataError card />;
  }

  return (
    <div
      className={`${baseClass} ${selectedPlatform ? "single-platform" : ""}`}
      style={opacity}
    >
      {renderCounts()}
    </div>
  );
};

export default HostsSummary;

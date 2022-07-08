import React, { useEffect } from "react";
import paths from "router/paths";

import { ILabelSummary } from "interfaces/label";
import { PLATFORM_NAME_TO_LABEL_NAME } from "utilities/constants";

import DataError from "components/DataError";

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
    <div className={`${baseClass}__tile mac-tile`}>
      <div className={`${baseClass}__tile-icon`}>
        <img src={MacIcon} alt="mac icon" id="mac-icon" />
      </div>
      <div>
        <div className={`${baseClass}__tile-count mac-count`}>{macCount}</div>
        <div className={`${baseClass}__tile-description`}>macOS hosts</div>
      </div>
    </div>
  );

  const renderWindowsCount = () => (
    <div className={`${baseClass}__tile windows-tile`}>
      <div className={`${baseClass}__tile-icon`}>
        <img src={WindowsIcon} alt="windows icon" id="windows-icon" />
      </div>
      <div>
        <div className={`${baseClass}__tile-count windows-count`}>
          {windowsCount}
        </div>
        <div className={`${baseClass}__tile-description`}>Windows hosts</div>
      </div>
    </div>
  );

  const renderLinuxCount = () => (
    <div className={`${baseClass}__tile linux-tile`}>
      <div className={`${baseClass}__tile-icon`}>
        <img src={LinuxIcon} alt="linux icon" id="linux-icon" />
      </div>
      <div>
        <div className={`${baseClass}__tile-count linux-count`}>
          {linuxCount}
        </div>
        <div className={`${baseClass}__tile-description`}>Linux hosts</div>
      </div>
    </div>
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

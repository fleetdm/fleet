import React, { useState, useEffect } from "react";
import { useQuery } from "react-query";
import { ILabel } from "interfaces/label";
import paths from "router/paths";

import { PLATFORM_NAME_TO_LABEL_NAME } from "utilities/constants";
import hostCountAPI from "services/entities/host_count";
import labelsAPI from "services/entities/labels";

import WindowsIcon from "../../../../../assets/images/icon-windows-48x48@2x.png";
import LinuxIcon from "../../../../../assets/images/icon-linux-48x48@2x.png";
import MacIcon from "../../../../../assets/images/icon-mac-48x48@2x.png";

const baseClass = "hosts-summary";

interface IHostSummaryProps {
  currentTeamId: number | undefined;
  macCount: string | undefined;
  windowsCount: string | undefined;
  isLoadingHostsSummary: boolean;
  showHostsUI: boolean;
  selectedPlatform: string;
  setTotalCount: (count: string | undefined) => void;
  setActionURL?: (url: string) => void;
}

interface ILabelsResponse {
  labels: ILabel[];
}

interface IHostCountResponse {
  count: number;
}

const HostsSummary = ({
  currentTeamId,
  macCount,
  windowsCount,
  isLoadingHostsSummary,
  showHostsUI,
  selectedPlatform,
  setTotalCount,
  setActionURL,
}: IHostSummaryProps): JSX.Element => {
  const { MANAGE_HOSTS } = paths;
  const [linuxCount, setLinuxCount] = useState<string | undefined>();

  const getLabel = (
    labelString: string,
    labels: ILabel[]
  ): ILabel | undefined => {
    return Object.values(labels).find((label: ILabel) => {
      return label.label_type === "builtin" && label.name === labelString;
    });
  };

  const { data: labels } = useQuery<ILabelsResponse, Error, ILabel[]>(
    ["labels", currentTeamId, selectedPlatform],
    () => labelsAPI.loadAll(),
    {
      select: (data: ILabelsResponse) => data.labels,
    }
  );

  useQuery<IHostCountResponse, Error, number>(
    ["linux host count", currentTeamId, selectedPlatform, macCount, windowsCount],
    () => {
      const linuxLabel = getLabel("All Linux", labels || []);
      return (
        hostCountAPI.load({
          selectedLabels: [`labels/${linuxLabel?.id}`],
          teamId: currentTeamId,
        }) || { count: 0 }
      );
    },
    {
      select: (data: IHostCountResponse) => data.count,
      enabled: !!labels,
      onSuccess: (data: number) => {
        setLinuxCount(data.toLocaleString("en-US"));

        // after we get the linux count, we can
        // determine which count to use based on the platform
        switch (selectedPlatform) {
          case "darwin":
            setTotalCount(macCount);
            break;
          case "windows":
            setTotalCount(windowsCount);
            break;
          case "linux":
            setTotalCount(data.toLocaleString("en-US"));
            break;
          default:
            // will be set in the parent to the server's total
            setTotalCount(undefined);
            break;
        }
      },
    }
  );

  // build the manage hosts URL
  useEffect(() => {
    if (labels) {
      let hostsURL = MANAGE_HOSTS;

      // platform must go first since it's a URL slug rather than params
      if (selectedPlatform) {
        const labelValue =
          PLATFORM_NAME_TO_LABEL_NAME[
            selectedPlatform as keyof typeof PLATFORM_NAME_TO_LABEL_NAME
          ];
        hostsURL += `/${getLabel(labelValue, labels)?.slug}`;
      }

      if (currentTeamId) {
        hostsURL += `/?team_id=${currentTeamId}`;
      }

      setActionURL && setActionURL(hostsURL);
    }
  }, [labels, selectedPlatform, currentTeamId]);

  // Renders opaque information as host information is loading
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

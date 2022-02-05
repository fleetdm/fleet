import React, { useState } from "react";
import { useQuery } from "react-query";
import { ILabel } from "interfaces/label";

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
}: IHostSummaryProps): JSX.Element => {
  const [linuxCount, setLinuxCount] = useState<string | undefined>();

  const getLabel = (labelString: string, labels: ILabel[]) => {
    return Object.values(labels).filter((label: ILabel) => {
      return label.label_type === "builtin" && label.name === labelString;
    });
  };
  const { data: labels } = useQuery<ILabelsResponse, Error, ILabel[]>(
    ["labels"],
    () => labelsAPI.loadAll(),
    {
      select: (data: ILabelsResponse) => data.labels,
    }
  );

  useQuery<IHostCountResponse, Error, number>(
    ["linux host count", currentTeamId, selectedPlatform],
    () => {
      const linuxLabel = getLabel("All Linux", labels || []);
      return (
        hostCountAPI.load({
          selectedLabels: [`labels/${linuxLabel[0].id}`],
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

  // Renders opaque information as host information is loading
  let opacity = { opacity: 0 };
  if (showHostsUI) {
    opacity = isLoadingHostsSummary ? { opacity: 0.4 } : { opacity: 1 };
  }

  const renderMacTile = () => (
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

  const renderWindowsTile = () => (
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

  const renderLinuxTile = () => (
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

  const renderTiles = () => {
    switch (selectedPlatform) {
      case "darwin":
        return renderMacTile();
      case "windows":
        return renderWindowsTile();
      case "linux":
        return renderLinuxTile();
      default:
        return (
          <>
            {renderMacTile()}
            {renderWindowsTile()}
            {renderLinuxTile()}
          </>
        );
    }
  };

  return (
    <div
      className={`${baseClass} ${selectedPlatform ? "single-platform" : ""}`}
      style={opacity}
    >
      {renderTiles()}
    </div>
  );
};

export default HostsSummary;

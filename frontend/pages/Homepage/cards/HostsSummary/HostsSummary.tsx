import React, { useCallback, useEffect, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { reduce } from "lodash";
import { ILabel } from "interfaces/label";
// @ts-ignore
import { getLabels } from "redux/nodes/components/ManageHostsPage/actions";

import hostCountAPI, {
  IHostCountLoadOptions,
} from "services/entities/host_count";

import WindowsIcon from "../../../../../assets/images/icon-windows-48x48@2x.png";
import LinuxIcon from "../../../../../assets/images/icon-linux-48x48@2x.png";
import MacIcon from "../../../../../assets/images/icon-mac-48x48@2x.png";

const baseClass = "hosts-summary";

interface IHostsSummaryProps {
  currentTeamId: number | undefined;
}
interface IRootState {
  entities: {
    labels: {
      isLoading: boolean;
      data: {
        [id: number]: ILabel;
      };
    };
  };
}

const PLATFORM_STRINGS = {
  macOS: "macOS",
  windows: "MS Windows",
  linux: "All Linux",
};

const HostsSummary = ({ currentTeamId }: IHostsSummaryProps): JSX.Element => {
  console.log("currentTeamId", currentTeamId);
  const dispatch = useDispatch();

  useEffect(() => {
    dispatch(getLabels());
  }, []);

  const [macCount, setMacCount] = useState<string | undefined>();
  const [windowsCount, setWindowsCount] = useState<string | undefined>();
  const [linuxCount, setLinuxCount] = useState<string | undefined>();

  const labels = useSelector((state: IRootState) => state.entities.labels.data);

  if (!currentTeamId) {
    const allTeamsHostCount = useCallback(() => {
      // Builtin labels from state populate os counts
      const getAllTeamsCount = (platformTitles: string) => {
        const count = reduce(
          Object.values(labels),
          (total, label) => {
            return label.label_type === "builtin" &&
              platformTitles === label.name &&
              label.count
              ? total + label.count
              : total;
          },
          0
        );
        return count;
      };

      setMacCount(
        getAllTeamsCount(PLATFORM_STRINGS.macOS).toLocaleString("en-US")
      );
      setWindowsCount(
        getAllTeamsCount(PLATFORM_STRINGS.windows).toLocaleString("en-US")
      );
      setLinuxCount(
        getAllTeamsCount(PLATFORM_STRINGS.linux).toLocaleString("en-US")
      );
    }, [currentTeamId]);

    allTeamsHostCount();
  } else {
    const teamHostCount = useCallback(() => {
      const macOsLabel = Object.values(labels).filter((label: ILabel) => {
        return label.label_type === "builtin" && label.name === "macOS";
      });

      const windowsLabel = Object.values(labels).filter((label: ILabel) => {
        return label.label_type === "builtin" && label.name === "MS Windows";
      });

      const linuxLabel = Object.values(labels).filter((label: ILabel) => {
        return label.label_type === "builtin" && label.name === "All Linux";
      });

      const retrieveHostCount = async () => {
        try {
          const { count: returnedTeamMacCount } = await hostCountAPI.load({
            selectedLabels: [`labels/${macOsLabel[0].id}`],
            teamId: currentTeamId,
          });
          const { count: returnedTeamWindowsCount } = await hostCountAPI.load({
            selectedLabels: [`labels/${windowsLabel[0].id}`],
            teamId: currentTeamId,
          });
          const { count: returnedTeamLinuxCount } = await hostCountAPI.load({
            selectedLabels: [`labels/${linuxLabel[0].id}`],
            teamId: currentTeamId,
          });

          setMacCount(returnedTeamMacCount.toLocaleString("en-US"));
          setWindowsCount(returnedTeamWindowsCount.toLocaleString("en-US"));
          setLinuxCount(returnedTeamLinuxCount.toLocaleString("en-US"));
        } catch (error) {
          console.error(error);
        }

        retrieveHostCount();
      };
    }, [currentTeamId]);

    teamHostCount();
  }
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__tile mac-tile`}>
        <div className={`${baseClass}__tile-icon`}>
          <img src={MacIcon} alt="mac icon" id="mac-icon" />
        </div>
        <div>
          <div className={`${baseClass}__tile-count mac-count`}>{macCount}</div>
          <div className={`${baseClass}__tile-description`}>macOS hosts</div>
        </div>
      </div>
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
    </div>
  );
};

export default HostsSummary;

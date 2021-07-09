import React from "react";
import { useDispatch, useSelector } from "react-redux";

// Incorporate this
import { ILabel } from "interfaces/label";

// @ts-ignore
import { getLabels } from "redux/nodes/components/ManageHostsPage/actions";

import WindowsIcon from "../../../../../assets/images/icon-windows-48x48@2x.png";
import LinuxIcon from "../../../../../assets/images/icon-linux-48x48@2x.png";
import MacIcon from "../../../../../assets/images/icon-mac-48x48@2x.png";

const baseClass = "host-summary";

interface RootState {
  entities: {
    labels: {
      isLoading: boolean;
      data: { [id: number]: any };
    };
  };
}

const HostSummary = (): JSX.Element => {
  // TODO: Get labels to load everytime into state
  const dispatch = useDispatch();
  dispatch(getLabels);

  const labels = useSelector((state: RootState) => state.entities.labels.data);

  const numberWithCommas = (x: number): string => {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  };

  const createMacCount = () => {
    let macCount = 0;
    if (labels[7]) {
      macCount = labels[7]["count"];
    }

    return numberWithCommas(macCount);
  };

  const createWindowsCount = () => {
    let windowsCount = 0;
    if (labels[10]) {
      windowsCount = labels[10]["count"];
    }

    return numberWithCommas(windowsCount);
  };

  const createLinuxCount = () => {
    let linuxCount = 0;
    if (labels[8] && labels[9]) {
      linuxCount = labels[8]["count"] + labels[9]["count"];
    }

    return numberWithCommas(linuxCount);
  };

  console.log(createMacCount());
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__tiles`}>
        <div className={`${baseClass}__tile mac-tile`}>
          <div className={`${baseClass}__tile-icon`}>
            <img src={MacIcon} alt="mac icon" id="mac-icon" />
          </div>
          <div className={`${baseClass}__tile-count mac-count`}>
            {createMacCount()}
          </div>
          <div className={`${baseClass}__tile-description`}>macOS hosts</div>
        </div>
        <div className={`${baseClass}__tile windows-tile`}>
          <div className={`${baseClass}__tile-icon`}>
            <img src={WindowsIcon} alt="windows icon" id="windows-icon" />
          </div>
          <div className={`${baseClass}__tile-count windows-count`}>
            {createWindowsCount()}
          </div>
          <div className={`${baseClass}__tile-description`}>Windows hosts</div>
        </div>
        <div className={`${baseClass}__tile linux-tile`}>
          <div className={`${baseClass}__tile-icon`}>
            <img src={LinuxIcon} alt="linux icon" id="linux-icon" />
          </div>
          <div className={`${baseClass}__tile-count linux-count`}>
            {createLinuxCount()}
          </div>
          <div className={`${baseClass}__tile-description`}>Linux hosts</div>
        </div>
      </div>
    </div>
  );
};

export default HostSummary;

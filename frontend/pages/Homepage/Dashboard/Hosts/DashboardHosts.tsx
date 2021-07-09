import React, { useState, useCallback } from "react"; //, { useEffect }
import { useDispatch, useSelector } from "react-redux";

import { ILabel } from "interfaces/label";

// @ts-ignore
import { getLabels } from "redux/nodes/components/ManageHostsPage/actions";
import { Link } from "react-router";
import { IUser } from "interfaces/user";
import WindowsIcon from "../../../../../assets/images/icon-windows-48x48@2x.png";
import LinuxIcon from "../../../../../assets/images/icon-linux-48x48@2x.png";
import MacIcon from "../../../../../assets/images/icon-mac-48x48@2x.png";

const baseClass = "dashboard-hosts";

interface RootState {
  entities: {
    labels: {
      isLoading: boolean;
      data: { [id: number]: ILabel };
    };
  };
}

const DashboardHosts = (): JSX.Element => {
  const dispatch = useDispatch();
  dispatch(getLabels);

  const labels = useSelector((state: RootState) => state.entities.labels.data);

  console.log(labels);

  const numberWithCommas = (x: number): string => {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  };

  const createMacCount = () => {
    const macCount = 15430;
    return numberWithCommas(macCount);
  };

  const createWindowsCount = () => {
    const windowsCount = 102343;
    return numberWithCommas(windowsCount);
  };

  const createLinuxCount = () => {
    const windowsCount = 7384;
    return numberWithCommas(windowsCount);
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

export default DashboardHosts;

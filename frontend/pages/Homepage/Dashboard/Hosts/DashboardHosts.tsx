import React, { useState, useCallback } from "react"; //, { useEffect }
import { useDispatch, useSelector } from "react-redux";

import { Link } from "react-router";
import { IUser } from "interfaces/user";
import WindowsIcon from "../../../../../assets/images/icon-windows-48x48@2x.png";
import LinuxIcon from "../../../../../assets/images/icon-windows-48x48@2x.png";
import MacIcon from "../../../../../assets/images/icon-windows-48x48@2x.png";

const baseClass = "dashboard-hosts";

interface RootState {
  auth: {
    user: IUser;
  };
  app: {
    config: {
      org_name: string;
    };
  };
}

const DashboardHosts = (): JSX.Element => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__tiles`}>
        <div className={`${baseClass}__tile mac-tile`}>Tile 1, 2, 3</div>
        <div className={`${baseClass}__tile windows-tile`}>Tile 1, 2, 3</div>
        <div className={`${baseClass}__tile linux-tile`}>Tile 1, 2, 3</div>
      </div>
    </div>
  );
};

export default DashboardHosts;

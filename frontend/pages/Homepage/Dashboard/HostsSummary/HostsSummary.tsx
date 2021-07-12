import React from "react";
import { useDispatch, useSelector } from "react-redux";

// @ts-ignore
import { getLabels } from "redux/nodes/components/ManageHostsPage/actions";

import WindowsIcon from "../../../../../assets/images/icon-windows-48x48@2x.png";
import LinuxIcon from "../../../../../assets/images/icon-linux-48x48@2x.png";
import MacIcon from "../../../../../assets/images/icon-mac-48x48@2x.png";

const baseClass = "hosts-summary";

interface IRootState {
  entities: {
    labels: {
      isLoading: boolean;
      data: {
        [id: number]: {
          count: number;
        };
      };
    };
  };
}

const HostsSummary = (): JSX.Element => {
  const dispatch = useDispatch();
  dispatch(getLabels());

  const labels = useSelector((state: IRootState) => state.entities.labels.data);

  const macCount = labels[7] ? labels[7].count.toLocaleString("en-US") : "";
  const windowsCount = labels[10]
    ? labels[10].count.toLocaleString("en-US")
    : "";
  const linuxCount =
    labels[8] && labels[9]
      ? (labels[8].count + labels[9].count + labels[11].count).toLocaleString(
          "en-US"
        )
      : "";

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__tiles`}>
        <div className={`${baseClass}__tile mac-tile`}>
          <div className={`${baseClass}__tile-icon`}>
            <img src={MacIcon} alt="mac icon" id="mac-icon" />
          </div>
          <div className={`${baseClass}__tile-count mac-count`}>{macCount}</div>
          <div className={`${baseClass}__tile-description`}>macOS hosts</div>
        </div>
        <div className={`${baseClass}__tile windows-tile`}>
          <div className={`${baseClass}__tile-icon`}>
            <img src={WindowsIcon} alt="windows icon" id="windows-icon" />
          </div>
          <div className={`${baseClass}__tile-count windows-count`}>
            {windowsCount}
          </div>
          <div className={`${baseClass}__tile-description`}>Windows hosts</div>
        </div>
        <div className={`${baseClass}__tile linux-tile`}>
          <div className={`${baseClass}__tile-icon`}>
            <img src={LinuxIcon} alt="linux icon" id="linux-icon" />
          </div>
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

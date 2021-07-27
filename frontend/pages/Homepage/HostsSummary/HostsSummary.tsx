import React, { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { filter, isEmpty, reduce } from "lodash";
// @ts-ignore
import { getLabels } from "redux/nodes/components/ManageHostsPage/actions";
import WindowsIcon from "../../../../assets/images/icon-windows-48x48@2x.png";
import LinuxIcon from "../../../../assets/images/icon-linux-48x48@2x.png";
import MacIcon from "../../../../assets/images/icon-mac-48x48@2x.png";

const baseClass = "hosts-summary";

interface IRootState {
  entities: {
    labels: {
      isLoading: boolean;
      data: {
        [id: number]: {
          count: number;
          name: string;
          label_type: string;
        };
      };
    };
  };
}

const PLATFORM_STRINGS = [
  "macOS",
  "MS Windows",
  "Red Hat Linux",
  "CentOS Linux",
  "Ubuntu Linux",
];

const HostsSummary = (): JSX.Element => {
  const dispatch = useDispatch();

  useEffect(() => {
    dispatch(getLabels());
  }, []);

  const labels = useSelector((state: IRootState) => state.entities.labels.data);

  // Builtin labels from state populate os counts
  const labelsWithIds: any = Object.keys(labels).map((id) => {
    const index = Number(id);
    return isNaN(index) ? null : labels[index];
  });
  const platformLabelsArray = labelsWithIds.filter((label: any) => {
    return (
      label &&
      label.label_type === "builtin" &&
      PLATFORM_STRINGS.includes(label.name)
    );
  });
  const getCount = (platformTitles: string[]) => {
    return reduce(
      platformLabelsArray,
      (acc: number, label: any) => {
        return platformTitles.includes(label.name) && !isEmpty(label.count)
          ? acc + label.count
          : acc;
      },
      0
    );
  };

  const macCount = getCount(["macOS"]);
  const windowsCount = getCount(["MS Windows"]);
  const linuxCount = getCount([
    "Red Hat Linux",
    "Ubuntu Linux",
    "CentOS Linux",
  ]);

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

import React from "react";
import Icon from "components/Icon";
import { ScheduledQueryablePlatform } from "interfaces/platform";

interface IPlatformCellProps {
  platforms: ScheduledQueryablePlatform[];
}

const baseClass = "platform-cell";

const ICONS: Record<string, ScheduledQueryablePlatform> = {
  darwin: "darwin",
  windows: "windows",
  linux: "linux",
};

const DISPLAY_ORDER: ScheduledQueryablePlatform[] = [
  "darwin",
  "windows",
  "linux",
];

const PlatformCell = ({ platforms }: IPlatformCellProps): JSX.Element => {
  let orderedList: ScheduledQueryablePlatform[] = [];
  orderedList = platforms.length
    ? // if no platforms, interpret as targeting all schedule-targetable platforms
      DISPLAY_ORDER.filter((platform) => platforms.includes(platform))
    : DISPLAY_ORDER;
  return (
    <span className={`${baseClass}__wrapper`} data-testid="icons">
      {orderedList.map((platform) => {
        return ICONS[platform] ? (
          <Icon
            className={`${baseClass}__icon`}
            name={ICONS[platform]}
            size="small"
            key={ICONS[platform]}
          />
        ) : null;
      })}
    </span>
  );
};

export default PlatformCell;

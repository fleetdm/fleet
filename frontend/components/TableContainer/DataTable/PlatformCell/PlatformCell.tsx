import React from "react";
import Icon from "components/Icon";
import { ScheduledQueryablePlatform } from "interfaces/platform";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface IPlatformCellProps {
  platforms: ScheduledQueryablePlatform[];
  queryIsScheduled?: boolean;
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

const PlatformCell = ({
  platforms,
  queryIsScheduled = false,
}: IPlatformCellProps): JSX.Element => {
  let orderedList: ScheduledQueryablePlatform[] = [];
  if (queryIsScheduled) {
    orderedList = platforms.length
      ? // if no platforms, interpret as targeting all schedule-targetable platforms
        DISPLAY_ORDER.filter((platform) => platforms.includes(platform))
      : DISPLAY_ORDER;
  }
  return (
    <span className={`${baseClass}__wrapper`} data-testid="icons">
      {orderedList.length ? (
        orderedList.map((platform) => {
          return ICONS[platform] ? (
            <Icon
              className={`${baseClass}__icon`}
              name={ICONS[platform]}
              size="small"
              key={ICONS[platform]}
            />
          ) : null;
        })
      ) : (
        <span className={`${baseClass}__muted`}>
          {DEFAULT_EMPTY_CELL_VALUE}
        </span>
      )}
    </span>
  );
};

export default PlatformCell;

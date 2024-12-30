import React from "react";
import Icon from "components/Icon";
import { QueryablePlatform } from "interfaces/platform";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface IPlatformCellProps {
  platforms: QueryablePlatform[];
}

const baseClass = "platform-cell";

const ICONS: Record<string, QueryablePlatform> = {
  darwin: "darwin",
  windows: "windows",
  linux: "linux",
  chrome: "chrome",
};

const DISPLAY_ORDER: QueryablePlatform[] = [
  "darwin",
  "windows",
  "linux",
  "chrome",
];

const PlatformCell = ({ platforms }: IPlatformCellProps): JSX.Element => {
  const orderedList = platforms.length
    ? DISPLAY_ORDER.filter((platform) => platforms.includes(platform))
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

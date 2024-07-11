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
  // "None",
  // "Invalid query",
];

const PlatformCell = ({ platforms }: IPlatformCellProps): JSX.Element => {
  const orderedList = DISPLAY_ORDER.filter((platform) =>
    platforms.includes(platform)
  );
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

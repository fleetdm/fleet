import React from "react";
import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { QueryablePlatform } from "interfaces/platform";

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
  // Chrome is supported for queries, but unsupported for scheduled queries
  // which are currently not supported in ChromeOS.
  "chrome",
];

const PlatformCell = ({ platforms }: IPlatformCellProps): JSX.Element => {
  let orderedList: QueryablePlatform[] = [];
  if (!platforms.length) {
    return <TextCell value="---" grey />;
  }
  orderedList = DISPLAY_ORDER.filter((platform) =>
    platforms.includes(platform)
  );
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

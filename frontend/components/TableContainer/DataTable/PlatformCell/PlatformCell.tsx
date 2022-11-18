import React from "react";
import Icon from "components/Icon";

interface IPlatformCellProps {
  value: string[];
}

const baseClass = "platform-cell";

const ICONS: Record<string, "darwin" | "linux" | "windows"> = {
  darwin: "darwin",
  linux: "linux",
  windows: "windows",
};

const DISPLAY_ORDER = [
  "darwin",
  "linux",
  "windows",
  // "None",
  // "Invalid query",
];

const PlatformCell = ({
  value: platforms,
}: IPlatformCellProps): JSX.Element => {
  const orderedList = DISPLAY_ORDER.filter((platform) =>
    platforms.includes(platform)
  );
  return (
    <span className={`${baseClass}__wrapper`}>
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
        <span className={`${baseClass}__muted`}>---</span>
      )}
    </span>
  );
};

export default PlatformCell;

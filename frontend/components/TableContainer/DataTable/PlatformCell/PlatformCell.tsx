import React from "react";
import AppleIcon from "../../../../../assets/images/icon-apple-dark-20x20@2x.png";
import LinuxIcon from "../../../../../assets/images/icon-linux-dark-20x20@2x.png";
import WindowsIcon from "../../../../../assets/images/icon-windows-dark-20x20@2x.png";

interface IPlatformCellProps {
  value: string[];
}

const baseClass = "platform-cell";

const ICONS: Record<string, string> = {
  darwin: AppleIcon,
  linux: LinuxIcon,
  windows: WindowsIcon,
};

const DISPLAY_ORDER = [
  "darwin",
  "linux",
  "windows",
  // "freebsd",
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
    <span>
      {orderedList.length ? (
        orderedList.map((platform) => {
          return ICONS[platform] ? (
            <img
              className={`${baseClass}__icon`}
              key={`platform-icon-${platform}`}
              alt={platform}
              src={ICONS[platform]}
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

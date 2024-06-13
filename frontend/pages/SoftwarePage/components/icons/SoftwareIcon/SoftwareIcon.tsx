import React from "react";
import getMatchedSoftwareIcon from "../";

const baseClass = "software-icon";

interface ISoftwareIconProps {
  name?: string;
  source?: string;
  size?: SoftwareIconSizes;
}

const SOFTWARE_ICON_SIZES: Record<string, string> = {
  medium: "24",
  meduim_large: "64", // TODO: rename this to large and update large to xlarge
  large: "96",
} as const;

type SoftwareIconSizes = keyof typeof SOFTWARE_ICON_SIZES;

const SoftwareIcon = ({
  name = "",
  source = "",
  size = "medium",
}: ISoftwareIconProps) => {
  const MatchedIcon = getMatchedSoftwareIcon({ name, source });
  return (
    <MatchedIcon
      width={SOFTWARE_ICON_SIZES[size]}
      height={SOFTWARE_ICON_SIZES[size]}
      viewBox="0 0 32 32"
      className={baseClass}
    />
  );
};

export default SoftwareIcon;

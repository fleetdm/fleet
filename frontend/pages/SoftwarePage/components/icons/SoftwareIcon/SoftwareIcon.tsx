import React, { ComponentType, SVGProps } from "react";
import {
  SOFTWARE_NAME_TO_ICON_MAP,
  SOFTWARE_SOURCE_TO_ICON_MAP,
  SOFTWARE_ICON_SIZES,
  SoftwareIconSizes,
  isSoftwareNameToIconException,
} from "../";

const baseClass = "software-icon";

interface ISoftwareIconProps {
  name?: string;
  source?: string;
  size?: SoftwareIconSizes;
}

const matchInMap = (
  map: Record<string, ComponentType<SVGProps<SVGSVGElement>>>,
  potentialKey?: string
) => {
  if (!potentialKey) {
    return null;
  }

  const sanitizedKey = potentialKey.trim().toLowerCase();
  const match = Object.entries(map).find(([namePrefix, icon]) => {
    if (sanitizedKey.startsWith(namePrefix)) {
      return icon;
    }
    return null;
  });

  return match ? match[1] : null;
};

const SoftwareIcon = ({
  name,
  source,
  size = "medium",
}: ISoftwareIconProps) => {
  let MatchedIcon: ComponentType<SVGProps<SVGSVGElement>> | null = null;

  if (name && !isSoftwareNameToIconException(name)) {
    // try to find a match for name
    MatchedIcon = matchInMap(SOFTWARE_NAME_TO_ICON_MAP, name);
  }

  // otherwise, try to find a match for source
  if (!MatchedIcon) {
    MatchedIcon = matchInMap(SOFTWARE_SOURCE_TO_ICON_MAP, source);
  }

  // default to 'package'
  if (!MatchedIcon) {
    MatchedIcon = SOFTWARE_SOURCE_TO_ICON_MAP.package;
  }

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

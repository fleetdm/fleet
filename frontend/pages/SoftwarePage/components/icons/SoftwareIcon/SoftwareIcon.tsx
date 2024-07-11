import React from "react";
import getMatchedSoftwareIcon from "../";
import classnames from "classnames";

const baseClass = "software-icon";

type SoftwareIconSizes = "small" | "medium" | "large";

interface ISoftwareIconProps {
  name?: string;
  source?: string;
  size?: SoftwareIconSizes;
  /** Accepts a url for a software icon image. */
  url?: string;
}

const SOFTWARE_ICON_SIZES: Record<SoftwareIconSizes, string> = {
  small: "24",
  medium: "64",
  large: "96",
};

const SoftwareIcon = ({
  name = "",
  source = "",
  size = "small",
  url,
}: ISoftwareIconProps) => {
  if (url) {
    const imgClasses = classnames(
      `${baseClass}__software-img`,
      `${baseClass}__software-img-${size}`
    );
    return (
      <div className={baseClass}>
        <img className={imgClasses} src={url} alt="" />
      </div>
    );
  }

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

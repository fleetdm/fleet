/** Renders operating system icons app-wide */

import React from "react";
import classnames from "classnames";

import { SOFTWARE_ICON_SIZES, SoftwareIconSizes } from "styles/var/icon_sizes";
import { getMatchedOsIcon } from "..";

const baseClass = "os-icon";

interface IOSIconProps {
  /** The operating system/distribution name (e.g. 'arch', 'ubuntu', 'windows', 'darwin', 'ios', etc). */
  name?: string;
  size?: SoftwareIconSizes;
  /** Accepts an image url to display for the OS icon image. */
  url?: string;
}

const OSIcon = ({ name = "", size = "small", url }: IOSIconProps) => {
  const classNames = classnames(baseClass, `${baseClass}__${size}`);

  // Use <img> if url is present
  if (url) {
    const imgClasses = classnames(
      `${baseClass}__os-img`,
      `${baseClass}__os-img-${size}`
    );
    return (
      <div className={classNames}>
        <img className={imgClasses} src={url} alt="" />
      </div>
    );
  }

  const MatchedIcon = getMatchedOsIcon({ name });

  return (
    <MatchedIcon
      width={SOFTWARE_ICON_SIZES[size]}
      height={SOFTWARE_ICON_SIZES[size]}
      viewBox="0 0 32 32"
      className={classNames}
    />
  );
};

export default OSIcon;

/** Renders software icons app-wide */

import React from "react";
import classnames from "classnames";
import { SOFTWARE_ICON_SIZES, SoftwareIconSizes } from "styles/var/icon_sizes";
import { getMatchedSoftwareIcon } from "../";

const baseClass = "software-icon";

interface ISoftwareIconProps {
  /** The software/application name */
  name?: string;
  /** Optional source string (e.g. "apps", "programs", etc) */
  source?: string;
  size?: SoftwareIconSizes;
  /** Accepts an image url to display for the software icon image. */
  url?: string | null;
}

const SoftwareIcon = ({
  name = "",
  source = "",
  size = "small",
  url,
}: ISoftwareIconProps) => {
  const classNames = classnames(baseClass, `${baseClass}__${size}`);
  // If we are given a url to render as the icon, we need to render it
  // differently than the svg icons. We will use an img tag instead with the
  // src set to the url.

  if (url) {
    const imgClasses = classnames(
      `${baseClass}__software-img`,
      `${baseClass}__software-img-${size}`
    );
    return (
      <div className={classNames}>
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
      className={classNames}
    />
  );
};

export default SoftwareIcon;

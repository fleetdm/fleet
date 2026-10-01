import React, { useState } from "react";

import { SOFTWARE_ICON_SIZES, SoftwareIconSizes } from "styles/var/icon_sizes";

import Package from "./Package";

/** Hand-authored icons stay inline SVG components. The ~1100 Fleet-maintained
 * app icons are raster images extracted to /assets, so they arrive as URLs. */
export type TMatchedIcon = string | React.FC<React.SVGProps<SVGSVGElement>>;

interface IMatchedIconProps {
  icon: TMatchedIcon;
  size: SoftwareIconSizes;
  className: string;
}

const MatchedIcon = ({ icon, size, className }: IMatchedIconProps) => {
  const px = SOFTWARE_ICON_SIZES[size];
  const [imgFailed, setImgFailed] = useState(false);

  if (typeof icon === "string" && !imgFailed) {
    return (
      <img
        className={className}
        src={icon}
        width={px}
        height={px}
        alt=""
        loading="lazy"
        onError={() => setImgFailed(true)}
      />
    );
  }

  const Icon = typeof icon === "string" ? Package : icon;
  return (
    <Icon width={px} height={px} viewBox="0 0 32 32" className={className} />
  );
};

export default MatchedIcon;

import React from "react";

import Icon from "components/Icon";
import TooltipTruncatedText from "components/TooltipTruncatedText";

import { OSUpdatesSupportedPlatform } from "../../OSUpdates";

const baseClass = "os-type-cell";

interface IOSTypeCellProps {
  platform: OSUpdatesSupportedPlatform;
  versionName: string;
}

const OSTypeCell = ({ platform, versionName }: IOSTypeCellProps) => {
  return (
    <div className={baseClass}>
      <Icon name={platform} />
      <div className={`${baseClass}__tooltip-wrapper`}>
        <TooltipTruncatedText
          value={versionName}
          className={`${baseClass}__inner-text`}
        />
      </div>
    </div>
  );
};

export default OSTypeCell;

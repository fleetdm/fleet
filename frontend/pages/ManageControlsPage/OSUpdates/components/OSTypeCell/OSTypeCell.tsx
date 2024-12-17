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
      <TooltipTruncatedText value={versionName} />
    </div>
  );
};

export default OSTypeCell;

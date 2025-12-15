import React from "react";

import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import { getPathWithQueryParams } from "utilities/url";
import { ANDROID_PLAY_STORE_URL } from "utilities/constants";

interface IAndroidLatestVersionWithTooltipProps {
  /** e.g. com.android.chrome, Component will build link URL using this ID */
  androidPlayStoreId: string;
}

/**  For Android Play Store apps version UI, we show "Latest" with tooltip
 * which links to the apps' play store */
const AndroidLatestVersionWithTooltip = ({
  androidPlayStoreId,
}: IAndroidLatestVersionWithTooltipProps) => {
  return (
    <TooltipWrapper
      tipContent={
        <span>
          See latest version on the{" "}
          <CustomLink
            text="Play Store"
            url={getPathWithQueryParams(ANDROID_PLAY_STORE_URL, {
              id: androidPlayStoreId,
            })}
            newTab
            variant="tooltip-link"
          />
        </span>
      }
    >
      <span>Latest</span>
    </TooltipWrapper>
  );
};

export default AndroidLatestVersionWithTooltip;

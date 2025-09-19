import React from "react";

import TooltipWrapper from "../TooltipWrapper";
import CustomLink from "../CustomLink";

export default function TooltipWrapperArchLinuxRolling() {
  return (
    <TooltipWrapper
      tipContent={
        <>
          Version follows a rolling release model.{" "}
          <CustomLink
            url="https://fleetdm.com/learn-more-about/arch-linux-rolling-release"
            variant="tooltip-link"
            text="Learn more"
            newTab
          />
        </>
      }
    >
      Rolling
    </TooltipWrapper>
  );
}

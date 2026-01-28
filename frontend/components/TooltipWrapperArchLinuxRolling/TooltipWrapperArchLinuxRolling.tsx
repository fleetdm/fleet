import React from "react";

import TooltipWrapper from "../TooltipWrapper";
import CustomLink from "../CustomLink";

interface TooltipWrapperArchLinuxRollingProps {
  capitalized?: boolean;
}

export default function TooltipWrapperArchLinuxRolling({
  capitalized = false,
}: TooltipWrapperArchLinuxRollingProps) {
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
      {capitalized ? "Rolling" : "rolling"}
    </TooltipWrapper>
  );
}

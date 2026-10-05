import React from "react";

import CustomLink from "../CustomLink";
import TooltipWrapper from "../TooltipWrapper";

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

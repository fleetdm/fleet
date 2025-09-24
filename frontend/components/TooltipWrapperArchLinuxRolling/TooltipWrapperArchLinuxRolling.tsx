import React from "react";

import TooltipWrapper from "../TooltipWrapper";
import CustomLink from "../CustomLink";

type Props = {
  capitalized?: boolean;
};

export default function TooltipWrapperArchLinuxRolling({
  capitalized = false,
}: Props) {
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

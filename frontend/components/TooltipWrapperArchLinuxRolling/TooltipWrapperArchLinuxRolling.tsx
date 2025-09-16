import React from "react";

import TooltipWrapper from "../TooltipWrapper";
import CustomLink from "../CustomLink";

export default function TooltipWrapperArchLinuxRolling() {
  return (
    <TooltipWrapper
      tipContent={
        <>
          Latest stable versions of most software by following a rolling release
          model.{" "}
          <CustomLink
            url="https://wiki.archlinux.org/title/Arch_Linux"
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

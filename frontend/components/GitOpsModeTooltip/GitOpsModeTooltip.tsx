import CustomLink from "components/CustomLink";
import TooltipWrapper, {
  ITooltipWrapper,
} from "components/TooltipWrapper/TooltipWrapper";
import { AppContext } from "context/app";
import React, { useContext } from "react";

interface IGitOpsModeTooltip {
  children: ITooltipWrapper["children"];
  position: ITooltipWrapper["position"];
  tipOffset: ITooltipWrapper["tipOffset"];
}

const GitOpsModeTooltip = ({
  children,
  position,
  tipOffset,
}: IGitOpsModeTooltip) => {
  const { config } = useContext(AppContext);
  const tipContent = (
    <>
      {config && (
        <>
          Manage in{" "}
          <CustomLink
            newTab
            text="YAML"
            variant="tooltip-link"
            url={config?.change_management.repository_url}
          />
          <br />
        </>
      )}
      (GitOps mode enabled)
    </>
  );
  return (
    <TooltipWrapper
      position={position}
      tipOffset={tipOffset}
      tipContent={tipContent}
    >
      {children}
    </TooltipWrapper>
  );
};

export default GitOpsModeTooltip;

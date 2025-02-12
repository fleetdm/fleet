import CustomLink from "components/CustomLink";
import TooltipWrapper, {
  ITooltipWrapper,
} from "components/TooltipWrapper/TooltipWrapper";
import { AppContext } from "context/app";
import React, { useContext } from "react";

interface IGitOpsModeTooltipWrapper {
  children: ITooltipWrapper["children"];
  position?: ITooltipWrapper["position"];
  tipOffset?: ITooltipWrapper["tipOffset"];
}

const baseClass = "gitops-mode-tooltip-wrapper";

const GitOpsModeTooltipWrapper = ({
  children,
  position,
  tipOffset,
}: IGitOpsModeTooltipWrapper) => {
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
      className={baseClass}
      position={position}
      tipOffset={tipOffset}
      tipContent={tipContent}
      underline={false}
      showArrow
    >
      {children}
    </TooltipWrapper>
  );
};

export default GitOpsModeTooltipWrapper;

import TooltipWrapper, {
  ITooltipWrapper,
} from "components/TooltipWrapper/TooltipWrapper";
import { AppContext } from "context/app";
import React, { useContext } from "react";
import { getGitOpsModeTipContent } from "utilities/helpers";

interface IGitOpsModeTooltipWrapper {
  renderChildren: (disableChildren?: boolean) => React.ReactNode;
  position?: ITooltipWrapper["position"];
  tipOffset?: ITooltipWrapper["tipOffset"];
  fixedPositionStrategy?: ITooltipWrapper["fixedPositionStrategy"];
}

const baseClass = "gitops-mode-tooltip-wrapper";

const GitOpsModeTooltipWrapper = ({
  position = "top",
  tipOffset,
  renderChildren,
  fixedPositionStrategy,
}: IGitOpsModeTooltipWrapper) => {
  const { config } = useContext(AppContext);
  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;
  const repoURL = config?.gitops.repository_url;

  if (!gitOpsModeEnabled) {
    return <>{renderChildren()}</>;
  }

  const tipContent = (
    <div className={`${baseClass}__tooltip-content`}>
      {repoURL && getGitOpsModeTipContent(repoURL)}
    </div>
  );

  return (
    <TooltipWrapper
      className={baseClass}
      position={position}
      tipOffset={tipOffset}
      tipContent={tipContent}
      underline={false}
      showArrow
      fixedPositionStrategy={fixedPositionStrategy}
    >
      {renderChildren(true)}
    </TooltipWrapper>
  );
};

export default GitOpsModeTooltipWrapper;

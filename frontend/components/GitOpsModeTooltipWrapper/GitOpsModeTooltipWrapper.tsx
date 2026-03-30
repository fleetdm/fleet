import TooltipWrapper, {
  ITooltipWrapper,
} from "components/TooltipWrapper/TooltipWrapper";
import useGitOpsMode from "hooks/useGitOpsMode";
import { IGitOpsExceptions } from "interfaces/config";
import React from "react";
import { getGitOpsModeTipContent } from "utilities/helpers";

interface IGitOpsModeTooltipWrapper {
  renderChildren: (disableChildren?: boolean) => React.ReactNode;
  position?: ITooltipWrapper["position"];
  tipOffset?: ITooltipWrapper["tipOffset"];
  fixedPositionStrategy?: ITooltipWrapper["fixedPositionStrategy"];
  // When specified, the wrapper checks the exception for this entity type.
  // If the entity is excepted, children remain enabled even in GitOps mode.
  entityType?: keyof IGitOpsExceptions;
}

const baseClass = "gitops-mode-tooltip-wrapper";

const GitOpsModeTooltipWrapper = ({
  position = "top",
  tipOffset,
  renderChildren,
  fixedPositionStrategy,
  entityType,
}: IGitOpsModeTooltipWrapper) => {
  const { gitOpsModeEnabled, repoURL } = useGitOpsMode(entityType);

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

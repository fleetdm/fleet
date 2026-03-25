import TooltipWrapper, {
  ITooltipWrapper,
} from "components/TooltipWrapper/TooltipWrapper";
import { AppContext } from "context/app";
import { GitOpsEntityType } from "interfaces/config";
import React, { useContext } from "react";
import { getGitOpsModeTipContent } from "utilities/helpers";

interface IGitOpsModeTooltipWrapper {
  renderChildren: (disableChildren?: boolean) => React.ReactNode;
  position?: ITooltipWrapper["position"];
  tipOffset?: ITooltipWrapper["tipOffset"];
  fixedPositionStrategy?: ITooltipWrapper["fixedPositionStrategy"];
  /** When specified, the wrapper checks the exception for this entity type.
   *  If the entity is excepted, children remain enabled even in GitOps mode. */
  entityType?: GitOpsEntityType;
}

const baseClass = "gitops-mode-tooltip-wrapper";

const GitOpsModeTooltipWrapper = ({
  position = "top",
  tipOffset,
  renderChildren,
  fixedPositionStrategy,
  entityType,
}: IGitOpsModeTooltipWrapper) => {
  const { config } = useContext(AppContext);
  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;
  const repoURL = config?.gitops.repository_url;

  // If GitOps mode is off, or the specified entity type is excepted,
  // render children without the tooltip and with no disabled state.
  if (
    !gitOpsModeEnabled ||
    (entityType && config?.gitops.exceptions[entityType])
  ) {
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

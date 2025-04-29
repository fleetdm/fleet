import CustomLink from "components/CustomLink";
import TooltipWrapper, {
  ITooltipWrapper,
} from "components/TooltipWrapper/TooltipWrapper";
import { AppContext } from "context/app";
import React, { useContext } from "react";

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
    // at this point repoURL will always be defined
    <div className={`${baseClass}__tooltip-content`}>
      {repoURL && (
        <span>
          Manage in{" "}
          <CustomLink newTab text="YAML" variant="tooltip-link" url={repoURL} />
          <br />
        </span>
      )}
      <span>(GitOps mode enabled)</span>
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

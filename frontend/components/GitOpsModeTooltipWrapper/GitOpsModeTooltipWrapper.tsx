import CustomLink from "components/CustomLink";
import TooltipWrapper, {
  ITooltipWrapper,
} from "components/TooltipWrapper/TooltipWrapper";
import { AppContext } from "context/app";
import React, { useContext } from "react";

interface IGitOpsModeTooltipWrapper {
  position?: ITooltipWrapper["position"];
  tipOffset?: ITooltipWrapper["tipOffset"];
  renderChildren: (disableChildren?: boolean) => React.ReactNode;
}

const baseClass = "gitops-mode-tooltip-wrapper";

const GitOpsModeTooltipWrapper = ({
  position = "top",
  tipOffset,
  renderChildren,
}: IGitOpsModeTooltipWrapper) => {
  const { config } = useContext(AppContext);
  const gomEnabled = config?.change_management.gitops_mode_enabled;
  const repoURL = config?.change_management.repository_url;

  if (!gomEnabled) {
    return <>{renderChildren()}</>;
  }

  const tipContent = (
    // at this point repoURL will always be defined
    <>
      {repoURL && (
        <>
          Manage in{" "}
          <CustomLink newTab text="YAML" variant="tooltip-link" url={repoURL} />
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
      {renderChildren(true)}
    </TooltipWrapper>
  );
};

export default GitOpsModeTooltipWrapper;

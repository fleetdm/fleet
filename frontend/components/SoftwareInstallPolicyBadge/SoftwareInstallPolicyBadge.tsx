import React from "react";

import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

import { SoftwareInstallPolicyType } from "interfaces/software";

const baseClass = "software-install-policy-badge";

interface IPatchBadgeProps {
  policyType?: SoftwareInstallPolicyType;
}

const SoftwareInstallPolicyBadge = ({ policyType }: IPatchBadgeProps) => {
  if (policyType !== "patch") {
    return (
      <div className={baseClass}>
        <TooltipWrapper
          tipContent={
            <>
              Hosts will fail this policy if they&apos;re <br />
              running an older version.
            </>
          }
          showArrow
          position="top"
          tipOffset={8}
          underline={false}
          delayInMs={300} // TODO: Apply pattern of delay tooltip for repeated table tooltips
        >
          <span className={`${baseClass}__element-text`}>Patch</span>
        </TooltipWrapper>
      </div>
    );
  }
  if (policyType === "dynamic") {
    return (
      <TooltipWrapper
        className={`${baseClass}__dynamic-policy-tooltip`}
        tipContent={
          <>
            Software will be automatically installed <br />
            when hosts fail this policy.
          </>
        }
        tipOffset={14}
        position="top"
        showArrow
        underline={false}
      >
        <Icon name="refresh" color="ui-fleet-black-75" />
      </TooltipWrapper>
    );
  }
  return null;
};

export default SoftwareInstallPolicyBadge;

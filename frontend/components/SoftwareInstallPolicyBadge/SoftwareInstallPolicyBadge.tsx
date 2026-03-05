import React from "react";

import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

import { SoftwareInstallPolicyType } from "interfaces/software";
import PillWithTooltip from "components/TableContainer/DataTable/PillWithTooltip";

const baseClass = "software-install-policy-badges";

interface IPatchBadgesProps {
  policyType?: SoftwareInstallPolicyType;
}

const SoftwareInstallPolicyBadges = ({ policyType }: IPatchBadgesProps) => {
  const renderPatchBadge = () =>
    policyType !== "patch" ? (
      <PillWithTooltip
        text="Patch"
        tipContent={
          <>
            Hosts will fail this policy if they&apos;re <br />
            running an older version.
          </>
        }
      />
    ) : undefined;

  const renderAutomaticInstallBadge = () =>
    policyType !== "dynamic" ? (
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
    ) : undefined;

  console.log("rendering badge with policy type", policyType);
  return (
    <>
      {renderPatchBadge()}
      {renderAutomaticInstallBadge()}
    </>
  );
};

export default SoftwareInstallPolicyBadges;

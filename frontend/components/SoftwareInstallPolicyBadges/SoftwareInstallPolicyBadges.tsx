import React from "react";

import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

import { SoftwareInstallPolicyTypeSet } from "interfaces/software";
import Tag from "components/Tag";

const baseClass = "software-install-policy-badges";

export const PATCH_TOOLTIP_CONTENT = (
  <>
    Hosts will fail this policy if they&apos;re <br />
    running an older version.
  </>
);
interface IPatchBadgesProps {
  policyType?: SoftwareInstallPolicyTypeSet;
}

const SoftwareInstallPolicyBadges = ({ policyType }: IPatchBadgesProps) => {
  const renderPatchBadge = () => (
    <Tag tooltip={PATCH_TOOLTIP_CONTENT} size="small">
      Patch
    </Tag>
  );

  const renderAutomaticInstallBadge = () => (
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

  return (
    <>
      {policyType?.has("patch") && renderPatchBadge()}
      {policyType?.has("dynamic") && renderAutomaticInstallBadge()}
    </>
  );
};

export default SoftwareInstallPolicyBadges;

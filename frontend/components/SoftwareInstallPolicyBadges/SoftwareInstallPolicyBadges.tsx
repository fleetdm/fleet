import React from "react";

import Icon from "components/Icon";

import { SoftwareInstallPolicyTypeSet } from "interfaces/software";
import PillBadge from "components/PillBadge";

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
    <PillBadge tipContent={PATCH_TOOLTIP_CONTENT}>Patch</PillBadge>
  );

  const renderAutomaticInstallBadge = () => (
    <Icon name="refresh" color="ui-fleet-black-75" />
  );

  return (
    <>
      {policyType?.has("patch") && renderPatchBadge()}
      {policyType?.has("dynamic") && renderAutomaticInstallBadge()}
    </>
  );
};

export default SoftwareInstallPolicyBadges;

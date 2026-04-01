import React from "react";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "critical-badge";

const CriticalPolicyBadge = () => {
  return (
    <div className={baseClass}>
      <TooltipWrapper
        tipContent="This policy has been marked as critical."
        showArrow
        position="top"
        tipOffset={8}
        underline={false}
        fixedPositionStrategy
      >
        <Icon
          className="critical-policy-icon"
          name="policy"
          size="small"
          color="ui-fleet-black-75"
        />
      </TooltipWrapper>
    </div>
  );
};

export default CriticalPolicyBadge;

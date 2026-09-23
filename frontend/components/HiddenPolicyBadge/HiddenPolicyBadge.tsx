import React from "react";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "hidden-policy-badge";

const HiddenPolicyBadge = () => {
  return (
    <div className={baseClass}>
      <TooltipWrapper
        tipContent="Hidden from end users"
        showArrow
        position="top"
        tipOffset={8}
        underline={false}
        fixedPositionStrategy
      >
        <Icon
          className="hidden-policy-icon"
          name="eye-slash"
          size="small"
          color="ui-fleet-black-33"
        />
      </TooltipWrapper>
    </div>
  );
};

export default HiddenPolicyBadge;

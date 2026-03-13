import React from "react";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

interface IIssuesIndicatorProps {
  totalIssuesCount?: number;
  failingPoliciesCount?: number;
  /** Premium only */
  criticalVulnerabilitiesCount?: number;
  tooltipPosition?: "top" | "bottom";
}

const IssuesIndicator = ({
  totalIssuesCount,
  failingPoliciesCount,
  criticalVulnerabilitiesCount,
  tooltipPosition = "top",
}: IIssuesIndicatorProps): JSX.Element => {
  return (
    <TooltipWrapper
      showArrow
      className="host-issue tooltip tooltip__tooltip-icon"
      tipContent={
        <span className="tooltip__tooltip-text">
          {!!criticalVulnerabilitiesCount &&
            `Critical vulnerabilities (${criticalVulnerabilitiesCount})`}
          {!!criticalVulnerabilitiesCount && !!failingPoliciesCount && <br />}
          {!!failingPoliciesCount &&
            `Failing policies (${failingPoliciesCount})`}
        </span>
      }
      position={tooltipPosition}
      underline={false}
    >
      <Icon name="error-outline" color="ui-fleet-black-50" /> {totalIssuesCount}
    </TooltipWrapper>
  );
};

export default IssuesIndicator;

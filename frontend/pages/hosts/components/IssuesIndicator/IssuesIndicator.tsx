import classnames from "classnames";
import React from "react";
import { PlacesType } from "react-tooltip-5";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

interface IIssuesIndicatorProps {
  totalIssuesCount?: number;
  failingPoliciesCount?: number;
  /** Premium only */
  criticalVulnerabilitiesCount?: number;
  tooltipPosition?: PlacesType;
  rowId?: string | number;
}

const baseClass = "issues-indicator";

const IssuesIndicator = ({
  totalIssuesCount,
  failingPoliciesCount,
  criticalVulnerabilitiesCount,
  tooltipPosition = "top",
  rowId,
}: IIssuesIndicatorProps): JSX.Element => {
  const classNames = classnames(baseClass, {
    [`${baseClass}--${rowId}`]: !!rowId,
  });
  const tipContent = (
    <>
      {!!criticalVulnerabilitiesCount &&
        `Critical vulnerabilities (${criticalVulnerabilitiesCount})`}
      {!!criticalVulnerabilitiesCount && !!failingPoliciesCount && <br />}
      {!!failingPoliciesCount && `Failing policies (${failingPoliciesCount})`}
    </>
  );

  return (
    <TooltipWrapper
      tipContent={tipContent}
      position={tooltipPosition}
      underline={false}
      showArrow
      className={classNames}
      tipOffset={8}
    >
      <Icon name="error-outline" color="ui-fleet-black-50" /> {totalIssuesCount}
    </TooltipWrapper>
  );
};

export default IssuesIndicator;

import React from "react";

import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";
import Icon from "components/Icon";

interface IIssuesIndicatorProps {
  totalIssuesCount?: number;
  failingPoliciesCount?: number;
  criticalVulnerabilitiesCount?: number;
}

const IssuesIndicator = ({
  totalIssuesCount,
  failingPoliciesCount,
  criticalVulnerabilitiesCount,
}: IIssuesIndicatorProps): JSX.Element => {
  return (
    <>
      <span
        className="host-issue tooltip tooltip__tooltip-icon"
        data-tip
        data-for="host-issue-count"
        data-tip-disable={false}
      >
        <Icon name="error-outline" color="ui-fleet-black-50" />{" "}
        {totalIssuesCount}
      </span>
      <ReactTooltip
        place="bottom"
        effect="solid"
        backgroundColor={COLORS["tooltip-bg"]}
        id="host-issue-count"
        data-html
      >
        <span className={`tooltip__tooltip-text`}>
          {criticalVulnerabilitiesCount &&
            `Critical vulnerabilities (${criticalVulnerabilitiesCount})`}
          <br />
          {failingPoliciesCount && `Failing policies (${failingPoliciesCount})`}
        </span>
      </ReactTooltip>
    </>
  );
};

export default IssuesIndicator;

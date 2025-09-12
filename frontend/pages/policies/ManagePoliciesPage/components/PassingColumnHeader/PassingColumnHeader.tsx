import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import { POLICY_STATUS_TO_INDICATOR_PARAMS } from "pages/hosts/details/cards/Policies/HostPoliciesTable/HostPoliciesTableConfig";
import React from "react";

interface IPassingColumnHeaderProps {
  isPassing: boolean;
}

const PassingColumnHeader = ({ isPassing }: IPassingColumnHeaderProps) => {
  const [indicatorStatus, displayText] = POLICY_STATUS_TO_INDICATOR_PARAMS[
    isPassing ? "pass" : "fail"
  ];
  return (
    <StatusIndicatorWithIcon value={displayText} status={indicatorStatus} />
  );
};

export default PassingColumnHeader;

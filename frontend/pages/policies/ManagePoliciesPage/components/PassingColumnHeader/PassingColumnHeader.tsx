import POLICY_STATUS_TO_INDICATOR_PARAMS from "components/policies/helpers";
import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
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

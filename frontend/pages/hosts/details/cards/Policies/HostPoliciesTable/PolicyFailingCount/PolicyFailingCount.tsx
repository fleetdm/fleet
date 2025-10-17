import { IHostPolicy } from "interfaces/policy";
import React from "react";

import InfoBanner from "components/InfoBanner";
import IconStatusMessage from "components/IconStatusMessage";

const baseClass = "policy-failing-count";

interface IPolicyFailingCountProps {
  policyList: IHostPolicy[];
  deviceUser?: boolean;
}
const PolicyFailingCount = ({
  policyList,
  deviceUser,
}: IPolicyFailingCountProps): JSX.Element | null => {
  const failCount = policyList.reduce((sum, policy) => {
    return policy.response === "fail" ? sum + 1 : sum;
  }, 0);

  return failCount ? (
    <InfoBanner className={baseClass} color="grey" borderRadius="xlarge">
      <IconStatusMessage
        iconName="error-outline"
        iconColor="ui-fleet-black-50"
        message={
          <span>
            <strong>
              {" "}
              This device is failing
              {failCount === 1 ? " 1 policy" : ` ${failCount} policies`}
            </strong>
            <br />
            Click a policy below to see if there are steps you can take to
            resolve the issue
            {failCount > 1 ? "s" : ""}.
            {deviceUser && " Once resolved, click “Refetch” above to confirm."}
          </span>
        }
      />
    </InfoBanner>
  ) : null;
};

export default PolicyFailingCount;

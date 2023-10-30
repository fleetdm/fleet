import { IHostPolicy } from "interfaces/policy";
import React from "react";

import Icon from "components/Icon/Icon";
import InfoBanner from "components/InfoBanner";

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
      <div className={`${baseClass}__count`}>
        <Icon name="error-outline" color="ui-fleet-black-50" />
        This device is failing
        {failCount === 1 ? " 1 policy" : ` ${failCount} policies`}
      </div>
      <p>
        Click a policy below to see if there are steps you can take to resolve
        the issue
        {failCount > 1 ? "s" : ""}.
        {deviceUser && " Once resolved, click “Refetch” above to confirm."}
      </p>
    </InfoBanner>
  ) : null;
};

export default PolicyFailingCount;

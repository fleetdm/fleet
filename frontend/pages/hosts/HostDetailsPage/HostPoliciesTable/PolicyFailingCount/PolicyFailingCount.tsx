import { IHostPolicy } from "interfaces/policy";
import React from "react";

import IssueIcon from "../../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";

const baseClass = "policy-failing-count";

const PolicyFailingCount = (policyProps: {
  policyList: IHostPolicy[];
}): JSX.Element | null => {
  const { policyList } = policyProps;

  const failCount = policyList.reduce((sum, policy) => {
    return policy.response === "fail" ? sum + 1 : sum;
  }, 0);

  return failCount ? (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__count`}>
        <img alt="Issue icon" src={IssueIcon} />
        This device is failing
        {failCount === 1 ? " 1 policy" : ` ${failCount} policies`}
      </div>
      <p>
        Click a policy below to see steps for resolving the failure
        {failCount > 1 ? "s" : ""}.
      </p>
    </div>
  ) : null;
};

export default PolicyFailingCount;

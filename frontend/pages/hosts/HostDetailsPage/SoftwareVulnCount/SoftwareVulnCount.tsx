import React from "react";

import { ISoftware } from "interfaces/software";
import IssueIcon from "../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";

const baseClass = "software-vuln-count";

const SoftwareVulnCount = (vulnProps: {
  softwareList: ISoftware[];
}): JSX.Element | null => {
  const { softwareList } = vulnProps;

  const vulnCount = softwareList.reduce((sum, software) => {
    return software.vulnerabilities
      ? sum + software.vulnerabilities.length
      : sum;
  }, 0);

  return vulnCount ? (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__count`}>
        <img alt="Issue icon" src={IssueIcon} />
        {vulnCount === 1
          ? "1 vulnerability detected"
          : `${vulnCount} vulnerabilities detected`}
      </div>
      <p>
        Click a vulnerable item below to see the associated Common
        Vulnerabilites and Exposures (CVEs).
      </p>
    </div>
  ) : null;
};

export default SoftwareVulnCount;

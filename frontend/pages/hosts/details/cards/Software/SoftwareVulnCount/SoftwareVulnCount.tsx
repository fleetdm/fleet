import React from "react";

import { ISoftware } from "interfaces/software";
import IssueIcon from "../../../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";

const baseClass = "software-vuln-count";

interface ISoftwareVulnCountProps {
  softwareList: ISoftware[];
  deviceUser?: boolean;
}

const SoftwareVulnCount = ({
  softwareList,
  deviceUser,
}: ISoftwareVulnCountProps): JSX.Element => {
  const vulnCount = softwareList.reduce((sum, software) => {
    return software.vulnerabilities?.length ? sum + 1 : sum;
  }, 0);
  return vulnCount ? (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__count`}>
        <img alt="Issue icon" src={IssueIcon} />
        {vulnCount === 1
          ? "1 software item with vulnerabilities detected"
          : `${vulnCount} software items with vulnerabilities detected`}
      </div>
    </div>
  ) : (
    <></>
  );
};

export default SoftwareVulnCount;

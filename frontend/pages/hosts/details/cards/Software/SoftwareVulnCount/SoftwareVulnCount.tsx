import React from "react";

import { ISoftware } from "interfaces/software";
import Icon from "components/Icon/Icon";
import InfoBanner from "components/InfoBanner";

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
    <InfoBanner className={baseClass} color="grey" borderRadius="xlarge">
      <div className={`${baseClass}__count`}>
        <Icon name="issue" />
        {vulnCount === 1
          ? "1 software item with vulnerabilities detected"
          : `${vulnCount} software items with vulnerabilities detected`}
      </div>
      {!deviceUser && (
        <p>
          Click a vulnerable item below to see the associated Common
          Vulnerabilites and Exposures (CVEs).
        </p>
      )}
    </InfoBanner>
  ) : (
    <></>
  );
};

export default SoftwareVulnCount;

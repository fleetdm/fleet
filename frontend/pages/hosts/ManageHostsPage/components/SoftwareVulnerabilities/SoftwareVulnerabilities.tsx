/* eslint-disable react/prop-types */
import React, { useState } from "react";

import { ISoftware } from "interfaces/software";

import CloseIcon from "../../../../../../assets/images/icon-close-fleet-black-16x16@2x.png";
import ExternalLinkIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";
import IssueIcon from "../../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";

interface ISoftwareVulnerabilitiesProps {
  software: ISoftware;
}

const baseClass = "software-vulnerabilities";

const SoftwareVulnerabilities = ({
  software,
}: ISoftwareVulnerabilitiesProps): JSX.Element | null => {
  const { name, version, vulnerabilities } = software;
  const count = vulnerabilities?.length;

  const [showVulnerabilities, setShowVulnerablities] = useState<boolean>(true);

  if (count && showVulnerabilities) {
    return (
      <div className={`${baseClass}`}>
        <div className={`${baseClass}__header`}>
          <div className={`${baseClass}__count`}>
            <img alt="Software vulnerabilities found" src={IssueIcon} />
            {`${
              count === 1 ? "1 vulnerability" : `${count} vulnerabilities`
            } detected ${name && version ? `for ${name}, ${version}` : ""}`}
          </div>
          <div className={`${baseClass}__ex`}>
            <button
              className="button button--unstyled"
              onClick={() => setShowVulnerablities(!showVulnerabilities)}
            >
              <img
                alt="Dismiss software vulnerabilities banner"
                src={CloseIcon}
              />
            </button>
          </div>
        </div>
        <div className={`${baseClass}__list`}>
          <ul>
            {vulnerabilities?.map((v) => {
              return (
                <li key={v.cve}>
                  Read more about{" "}
                  <a
                    href={v.details_link}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {v.cve} vulnerability{" "}
                    <img alt="External link" src={ExternalLinkIcon} />
                  </a>
                </li>
              );
            })}
          </ul>
        </div>
      </div>
    );
  }

  return null;
};
export default SoftwareVulnerabilities;

import React from "react";
import { uniqueId } from "lodash";
import ReactTooltip from "react-tooltip";

import TextCell from "components/TableContainer/DataTable/TextCell";
import { ISoftwareVulnerability } from "interfaces/software";
import { is } from "date-fns/locale";

const baseClass = "vulnerabilities-cell";

const generateVulnerabilitiesCell = (
  vulnerabilities: ISoftwareVulnerability[] | string[] | null
) => {
  if (vulnerabilities === null) {
    return <TextCell value="---" greyed />;
  }

  let text = "";
  let isGrayed = true;
  if (vulnerabilities.length === 0) {
    text = "---";
  } else if (vulnerabilities.length === 1) {
    isGrayed = false;
    text =
      typeof vulnerabilities[0] === "string"
        ? vulnerabilities[0]
        : vulnerabilities[0].cve;
  } else {
    text = `${vulnerabilities.length} vulnerabilities`;
  }

  return <TextCell value={text} greyed={isGrayed} />;
};

const generateVulnerabilitiesTooltip = (
  vulnerabilities: ISoftwareVulnerability[] | string[],
  tooltipId: string
) => {
  if (vulnerabilities.length <= 1) {
    return null;
  }

  return (
    <ReactTooltip
      effect="solid"
      backgroundColor="#3e4771"
      id={tooltipId}
      data-html
    >
      <ul className={`${baseClass}__vulnerability-list`}>
        {vulnerabilities.map((vulnerability) => {
          const text =
            typeof vulnerability === "string"
              ? vulnerability
              : vulnerability.cve;
          return <li>&bull; {text}</li>;
        })}
      </ul>
    </ReactTooltip>
  );
};
interface IVulnerabilitiesCellProps {
  vulnerabilities: ISoftwareVulnerability[] | string[] | null;
}

const VulnerabilitiesCell = ({
  vulnerabilities,
}: IVulnerabilitiesCellProps) => {
  const tooltipId = uniqueId();

  // only one vulnerability, no need for tooltip
  const cell = generateVulnerabilitiesCell(vulnerabilities);
  if (vulnerabilities === null || vulnerabilities.length <= 1) {
    return <>{cell}</>;
  }

  const versionTooltip = generateVulnerabilitiesTooltip(
    vulnerabilities,
    tooltipId
  );

  return (
    <>
      <div
        className={`${baseClass}__vulnerability-text-with-tooltip`}
        data-tip
        data-for={tooltipId}
      >
        {cell}
      </div>
      {versionTooltip}
    </>
  );
};

export default VulnerabilitiesCell;

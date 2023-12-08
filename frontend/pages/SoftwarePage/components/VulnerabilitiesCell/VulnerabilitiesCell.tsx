import TextCell from "components/TableContainer/DataTable/TextCell";
import { ISoftwareVulnerability } from "interfaces/software";
import React from "react";

const getSumOfVulnerabilities = (
  vulnerabilities: ISoftwareVulnerability[] | string[] | null
) => {
  if (vulnerabilities === null || vulnerabilities.length === 0) {
    return 0;
  }
  return vulnerabilities.length;
};

interface IVulnerabilitiesCellProps {
  vulnerabilities: ISoftwareVulnerability[] | string[] | null;
}

const VulnerabilitiesCell = ({
  vulnerabilities,
}: IVulnerabilitiesCellProps) => {
  const numVulnerabilities = getSumOfVulnerabilities(vulnerabilities);

  let text = "";
  if (numVulnerabilities === 0) {
    text = "---";
  } else if (numVulnerabilities === 1) {
    text = "1 vulnerability";
  } else {
    text = `${numVulnerabilities} vulnerabilities`;
  }

  return <TextCell value={text} greyed />;
};

export default VulnerabilitiesCell;

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import React from "react";

const baseClass = "org-name-cell";

interface IOrgNameCellProps {
  orgName: string;
  termsExpired: boolean;
}

const OrgNameCell = ({ orgName, termsExpired }: IOrgNameCellProps) => {
  const cellContent = termsExpired ? (
    <>
      <span>{orgName}</span> <Icon name="warning" />
    </>
  ) : (
    orgName
  );
  return <TextCell value={cellContent} className={baseClass} />;
};

export default OrgNameCell;

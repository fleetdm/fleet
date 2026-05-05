import React from "react";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "org-name-cell";

interface IOrgNameCellProps {
  orgName: string;
  termsExpired: boolean;
}

const OrgNameCell = ({ orgName, termsExpired }: IOrgNameCellProps) => {
  const cellContent = termsExpired ? (
    <TooltipWrapper
      showArrow
      underline={false}
      position="top"
      tipContent={
        <>
          The ABM terms have changed.
          <br />
          To accept terms, go to ABM.
        </>
      }
      className={`${baseClass}__tooltip-wrapper`}
    >
      <span>{orgName}</span> <Icon name="warning" />
    </TooltipWrapper>
  ) : (
    orgName
  );
  return <TextCell value={cellContent} className={baseClass} />;
};

export default OrgNameCell;

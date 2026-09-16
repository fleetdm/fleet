import React from "react";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import Tag from "components/Tag";
import TooltipWrapper from "components/TooltipWrapper";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

const baseClass = "org-name-cell";

interface IOrgNameCellProps {
  orgName: string;
  termsExpired: boolean;
  isDefault: boolean;
}

const OrgNameCell = ({
  orgName,
  termsExpired,
  isDefault,
}: IOrgNameCellProps) => {
  const name = termsExpired ? (
    <TooltipWrapper
      showArrow
      underline={false}
      position="top"
      tipContent="The AB terms have changed. To accept terms, go to AB."
      className={`${baseClass}__tooltip-wrapper`}
    >
      <span>{orgName}</span> <Icon name="warning" />
    </TooltipWrapper>
  ) : (
    orgName
  );

  const cellContent = isDefault ? (
    <>
      {name}{" "}
      <Tag
        size="xsmall"
        tooltip={
          <>
            Used when a manually enrolling device requests a token for a Managed
            Apple ID login.{" "}
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/default-ab-token`}
              text="Learn more"
              variant="tooltip-link"
              newTab
            />
          </>
        }
      >
        Default token
      </Tag>
    </>
  ) : (
    name
  );

  return <TextCell value={cellContent} className={baseClass} />;
};

export default OrgNameCell;

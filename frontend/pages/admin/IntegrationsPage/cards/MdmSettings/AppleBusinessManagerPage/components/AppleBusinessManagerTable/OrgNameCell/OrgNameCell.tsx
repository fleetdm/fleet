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
            Restricting Managed Apple Account sign-in to managed hosts only
            works if the restriction is turned on in Apple Business.{" "}
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/default-ab-token`}
              text="Learn more"
              variant="tooltip-link"
              newTab
            />
          </>
        }
      >
        Default sign-in
      </Tag>
    </>
  ) : (
    name
  );

  return <TextCell value={cellContent} className={baseClass} />;
};

export default OrgNameCell;

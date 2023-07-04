import React from "react";

import classnames from "classnames";
import TooltipWrapper from "components/TooltipWrapper";

interface IHeaderCellProps {
  value: string | JSX.Element; // either a string or a TooltipWrapper
  isSortedDesc?: boolean;
  disableSortBy?: boolean;
  isLastColumn?: boolean;
}

const HeaderCell = ({
  value,
  isSortedDesc,
  disableSortBy,
  isLastColumn = false,
}: IHeaderCellProps): JSX.Element => {
  let sortArrowClass = "";
  if (isSortedDesc === undefined) {
    sortArrowClass = "";
  } else if (isSortedDesc) {
    sortArrowClass = "descending";
  } else {
    sortArrowClass = "ascending";
  }

  let lastColumnHeaderWithTooltipClass = "";
  if (
    typeof value !== "string" &&
    value.type === TooltipWrapper &&
    isLastColumn
  ) {
    lastColumnHeaderWithTooltipClass = "last-col-header-with-tip";
  }

  return (
    <div
      className={classnames(
        "header-cell",
        sortArrowClass,
        lastColumnHeaderWithTooltipClass
      )}
    >
      <span>{value}</span>
      {!disableSortBy && (
        <div className="sort-arrows">
          <span className="ascending-arrow" />
          <span className="descending-arrow" />
        </div>
      )}
    </div>
  );
};

export default HeaderCell;

import React from "react";

import classnames from "classnames";

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
  // Since value will only be a string or a TooltipWrapper, this checks for a TooltipWrapper
  // TODO - add better checking for this
  if (typeof value !== "string" && isLastColumn) {
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

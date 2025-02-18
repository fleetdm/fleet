import React, { ReactNode } from "react";

import classnames from "classnames";

interface IHeaderCellProps {
  value: ReactNode;
  isSortedDesc?: boolean;
  disableSortBy?: boolean;
  tootip?: ReactNode;
}

const HeaderCell = ({
  value,
  isSortedDesc,
  disableSortBy,
  tootip,
}: IHeaderCellProps): JSX.Element => {
  let sortArrowClass = "";
  if (isSortedDesc === undefined) {
    sortArrowClass = "";
  } else if (isSortedDesc) {
    sortArrowClass = "descending";
  } else {
    sortArrowClass = "ascending";
  }

  return (
    <div className={classnames("header-cell", sortArrowClass)}>
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

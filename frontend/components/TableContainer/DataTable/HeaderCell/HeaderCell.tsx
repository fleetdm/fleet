import React from "react";

interface IHeaderCellProps {
  value: string;
  isSortedDesc?: boolean;
}

const HeaderCell = (props: IHeaderCellProps): JSX.Element => {
  const { value, isSortedDesc } = props;

  let sortArrowClass = "";
  if (isSortedDesc === undefined) {
    sortArrowClass = "";
  } else if (isSortedDesc) {
    sortArrowClass = "descending";
  } else {
    sortArrowClass = "ascending";
  }

  return (
    <div className={`header-cell ${sortArrowClass}`}>
      <span>{value}</span>
      <div className="sort-arrows">
        <span className="ascending-arrow" />
        <span className="descending-arrow" />
      </div>
    </div>
  );
};

export default HeaderCell;

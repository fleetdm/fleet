import React from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";

const baseClass = "pagination";

export interface IPaginationProps {
  /**  Disable next page is usually passed through from api metadata, or on loading */
  disableNext?: boolean;
  /**  Disable prev page is usually passed through from api meta data, on page 0, or on loading */
  disablePrev?: boolean;
  onNextPage: () => void;
  onPrevPage: () => void;
  className?: string;
  /** UI Pattern: Hide pagination iff there's one page of results */
  hidePagination?: boolean;
}

const Pagination = ({
  disableNext,
  disablePrev,
  onNextPage,
  onPrevPage,
  className,
  hidePagination = false,
}: IPaginationProps) => {
  const classNames = classnames(baseClass, className);

  if (hidePagination) {
    return null;
  }

  return (
    <div className={classNames}>
      <Button
        variant="subdued"
        disabled={disablePrev}
        onClick={onPrevPage}
        className={`${baseClass}__pagination-button`}
        icon="chevron-left"
      >
        Previous
      </Button>
      <Button
        variant="subdued"
        disabled={disableNext}
        onClick={onNextPage}
        className={`${baseClass}__pagination-button`}
        icon="chevron-right"
        iconPosition="right"
      >
        Next
      </Button>
    </div>
  );
};

export default Pagination;

import React from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

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
        variant="inverse"
        disabled={disablePrev}
        onClick={onPrevPage}
        className={`${baseClass}__pagination-button`}
      >
        <Icon name="chevron-left" color="core-fleet-blue" /> Previous
      </Button>
      <Button
        variant="inverse"
        disabled={disableNext}
        onClick={onNextPage}
        className={`${baseClass}__pagination-button`}
      >
        Next <Icon name="chevron-right" color="core-fleet-blue" />
      </Button>
    </div>
  );
};

export default Pagination;

import React from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "pagination-new";

interface IPaginationProps {
  disableNext?: boolean;
  disablePrev?: boolean;
  onNextPage?: () => void;
  onPrevPage?: () => void;
  className?: string;
}

/**
 * This is the new pagination component that we will want to replace other pagination
 * components with. Going forward this should be the component used for pagination.
 */
const Pagination = ({
  disableNext,
  disablePrev,
  onNextPage,
  onPrevPage,
  className,
}: IPaginationProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      <Button
        variant="unstyled"
        disabled={disablePrev}
        onClick={onPrevPage}
        className={`${baseClass}__pagination-button`}
      >
        <Icon name="chevron-left" color="core-fleet-blue" /> Previous
      </Button>
      <Button
        variant="unstyled"
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

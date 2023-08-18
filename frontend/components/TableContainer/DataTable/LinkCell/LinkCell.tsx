// Utilizes Link over Button so we can right click links
import React from "react";

import { Link } from "react-router";
import classnames from "classnames";

interface ILinkCellProps {
  value: string | JSX.Element;
  path: string;
  className?: string;
  customOnClick?: (e: React.MouseEvent) => void;
  /** allows viewing overflow for tooltip */
  withTooltip?: boolean;
  title?: string;
}

const baseClass = "link-cell";

const LinkCell = ({
  value,
  path,
  className,
  customOnClick,
  withTooltip,
  title,
}: ILinkCellProps): JSX.Element => {
  const cellClasses = classnames(
    baseClass,
    className,
    withTooltip && "link-cell-tooltip"
  );

  const onClick = (e: React.MouseEvent): void => {
    customOnClick && customOnClick(e);
  };

  return (
    <Link className={cellClasses} to={path} onClick={onClick} title={title}>
      {value}
    </Link>
  );
};

export default LinkCell;

import React from "react";

// using browserHistory directly because "router"
// is difficult to pass as a prop
import { browserHistory, Link } from "react-router";
import classnames from "classnames";

import Button from "components/buttons/Button/Button";

interface ILinkCellProps {
  value: string | JSX.Element;
  path: string;
  title?: string;
  className?: string;
  customOnClick?: (e: React.MouseEvent) => void;
  /** allows viewing of tooltip */
  withTooltip?: boolean;
}

const baseClass = "link-cell";

const LinkCell = ({
  value,
  path,
  title,
  className,
  customOnClick,
  withTooltip,
}: ILinkCellProps): JSX.Element => {
  const cellClasses = classnames(
    baseClass,
    className,
    withTooltip && "link-cell-tooltip"
  );
  console.log("cellClassess", cellClasses);

  const onClick = (e: React.MouseEvent): void => {
    customOnClick && customOnClick(e);
    // browserHistory.push(path);
  };

  return (
    <Link className={cellClasses} to={path} onClick={onClick}>
      {value}
    </Link>
  );
  return (
    <Button
      className={`link-cell ${classes}`}
      onClick={onClick}
      variant="text-link"
      title={title}
    >
      {value}
    </Button>
  );
};

export default LinkCell;

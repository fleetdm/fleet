// Utilizes Link over Button so we can right click links
import React from "react";

import { Link } from "react-router";
import classnames from "classnames";
import TooltipWrapper from "components/TooltipWrapper";

interface ILinkCellProps {
  value: string | JSX.Element;
  path: string;
  className?: string;
  customOnClick?: (e: React.MouseEvent) => void;
  /** allows viewing overflow for tooltip */
  tooltipContent?: string | React.ReactNode;
  title?: string;
}

const baseClass = "link-cell";

const LinkCell = ({
  value,
  path,
  className,
  customOnClick,
  title,
  tooltipContent,
}: ILinkCellProps): JSX.Element => {
  const cellClasses = classnames(baseClass, className);

  const onClick = (e: React.MouseEvent): void => {
    customOnClick && customOnClick(e);
  };

  return tooltipContent ? (
    <TooltipWrapper
      className="link-cell-tooltip-wrapper"
      tipContent={tooltipContent}
    >
      <Link className={cellClasses} to={path} onClick={onClick} title={title}>
        {value}
      </Link>
    </TooltipWrapper>
  ) : (
    <Link
      className={cellClasses}
      to={path}
      onClick={customOnClick}
      title={title}
    >
      {value}
    </Link>
  );
};

export default LinkCell;

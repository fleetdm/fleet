// Utilizes Link over Button so we can right click links
import React from "react";

import { Link } from "react-router";
import classnames from "classnames";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedTextCell from "../TooltipTruncatedTextCell";

interface ILinkCellProps {
  value: string | JSX.Element;
  path?: string;
  className?: string;
  customOnClick?: (e: React.MouseEvent) => void;
  /** allows viewing overflow for tooltip */
  tooltipContent?: string | React.ReactNode;
  title?: string;
  /** Used to create TooltipTruncationText on link cell */
  tooltipTruncate?: boolean;
  /** Optionally add unstyled prefix before tooltip truncation */
  prefix?: JSX.Element;
  /** Optionally add unstyled suffix after tooltip truncation */
  suffix?: JSX.Element;
}

const baseClass = "link-cell";

const LinkCell = ({
  value,
  path = "",
  className,
  customOnClick,
  title,
  tooltipContent,
  tooltipTruncate = false,
  prefix,
  suffix,
}: ILinkCellProps): JSX.Element => {
  const cellClasses = classnames(baseClass, className);

  const onClick = (e: React.MouseEvent): void => {
    customOnClick && customOnClick(e);
  };

  if (tooltipTruncate)
    return (
      <Link
        className={cellClasses}
        to={path}
        onClick={customOnClick}
        title={title}
      >
        <TooltipTruncatedTextCell
          value={value}
          prefix={prefix}
          suffix={suffix}
        />
      </Link>
    );

  if (tooltipContent)
    return (
      <TooltipWrapper
        className="link-cell-tooltip-wrapper"
        tipContent={tooltipContent}
      >
        <Link className={cellClasses} to={path} onClick={onClick} title={title}>
          {value}
        </Link>
      </TooltipWrapper>
    );

  return (
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

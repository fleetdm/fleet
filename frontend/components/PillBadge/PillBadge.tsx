import React from "react";
import classnames from "classnames";

import TooltipWrapper from "components/TooltipWrapper";
import Tag from "components/Tag";

const baseClass = "pill-badge";

interface IPillBadgeProps {
  children: React.ReactNode;
  tipContent?: JSX.Element | string;
  className?: string;
  /** Default: "large" (28px). Use "small" (24px) within a table row. */
  size?: "large" | "small";
}

const PillBadge = ({
  children,
  tipContent,
  className,
  size,
}: IPillBadgeProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      <TooltipWrapper
        tipContent={tipContent}
        showArrow
        underline={false}
        position="top"
        tipOffset={12}
        delayInMs={300}
      >
        <Tag size={size}>{children}</Tag>
      </TooltipWrapper>
    </div>
  );
};

export default PillBadge;

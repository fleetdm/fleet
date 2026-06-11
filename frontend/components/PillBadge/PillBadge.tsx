import React from "react";
import classnames from "classnames";

import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "pill-badge";

interface IPillBadgeProps {
  children: React.ReactNode;
  tipContent?: JSX.Element | string;
  className?: string;
}

const PillBadge = ({ children, tipContent, className }: IPillBadgeProps) => {
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
        <span className={`${baseClass}__element`}>{children}</span>
      </TooltipWrapper>
    </div>
  );
};

export default PillBadge;

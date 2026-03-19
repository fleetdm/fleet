import React from "react";
import classnames from "classnames";

import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "pill-badge";

interface IPillBadge {
  children?: React.ReactNode;
  /** @deprecated Use children instead */
  text?: string;
  tipContent?: JSX.Element | string;
  className?: string;
}

const PillBadge = ({ children, text, tipContent, className }: IPillBadge) => {
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
        <span className={`${baseClass}__element`}>{children ?? text}</span>
      </TooltipWrapper>
    </div>
  );
};

export default PillBadge;

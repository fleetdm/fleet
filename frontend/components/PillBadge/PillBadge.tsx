import React from "react";
import classnames from "classnames";

import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "pill-badge";

interface IPillBadgeWithChildren {
  children: React.ReactNode;
  text?: never;
  tipContent?: JSX.Element | string;
  className?: string;
}

interface IPillBadgeWithText {
  children?: never;
  /** @deprecated Use children instead */
  text: string;
  tipContent?: JSX.Element | string;
  className?: string;
}

type IPillBadge = IPillBadgeWithChildren | IPillBadgeWithText;

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

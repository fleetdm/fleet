import React from "react";

import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "pill-badge";

interface IPillBadge {
  text: string;
  tipContent?: JSX.Element | string;
}

const PillBadge = ({ text, tipContent }: IPillBadge) => {
  return (
    <div className={baseClass}>
      <TooltipWrapper
        tipContent={tipContent}
        showArrow
        underline={false}
        position="top"
        tipOffset={12}
        delayInMs={300}
      >
        <span className={`${baseClass}__element-text`}>{text}</span>
      </TooltipWrapper>
    </div>
  );
};

export default PillBadge;

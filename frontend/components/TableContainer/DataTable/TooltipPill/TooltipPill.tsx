import React from "react";

import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "tooltip-pill";

interface ITooltipPill {
  text: string;
  tipContent?: JSX.Element | string;
}

const TooltipPill = ({ text, tipContent }: ITooltipPill) => {
  return (
    <div className={`${baseClass}__`}>
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

export default TooltipPill;

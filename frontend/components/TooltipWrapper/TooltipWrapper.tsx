import React from "react";

interface ITooltipWrapperProps {
  children: string;
  tipText: string;
}

const baseClass = "component__tooltip-wrapper";

const TooltipWrapper = ({
  children,
  tipText,
}: ITooltipWrapperProps): JSX.Element => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__element`}>
        {children}
        <div className={`${baseClass}__underline`} data-text={children} />
      </div>
      <div className={`${baseClass}__tip-text`}>{tipText}</div>
    </div>
  );
};

export default TooltipWrapper;

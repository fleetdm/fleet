import React from "react";

interface ITooltipWrapperProps {
  children: string;
  tipContent: string;
}

const baseClass = "component__tooltip-wrapper";

const TooltipWrapper = ({
  children,
  tipContent,
}: ITooltipWrapperProps): JSX.Element => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__element`}>
        {children}
        <div className={`${baseClass}__underline`} data-text={children} />
      </div>
      <div
        className={`${baseClass}__tip-text`}
        dangerouslySetInnerHTML={{ __html: tipContent }}
      />
    </div>
  );
};

export default TooltipWrapper;

import React from "react";

interface ITooltipWrapperProps {
  children: string;
  tipContent: string;
  position?: "top" | "bottom";
  tipDelay?: boolean;
}

const baseClass = "component__tooltip-wrapper";

const TooltipWrapper = ({
  children,
  tipContent,
  position = "bottom",
  tipDelay,
}: ITooltipWrapperProps): JSX.Element => {
  const tipClass = tipDelay
    ? `${baseClass}__tip-text delayed-tip`
    : `${baseClass}__tip-text`;

  return (
    <div className={baseClass} data-position={position}>
      <div className={`${baseClass}__element`}>
        {children}
        <div className={`${baseClass}__underline`} data-text={children} />
      </div>
      <div
        className={tipClass}
        dangerouslySetInnerHTML={{ __html: tipContent }}
      />
    </div>
  );
};

export default TooltipWrapper;

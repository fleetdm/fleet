import classnames from "classnames";
import React from "react";

interface ITooltipWrapperProps {
  children: string;
  tipContent: string;
  position?: "top" | "bottom";
  isDelayed?: boolean;
  className?: string;
}

const baseClass = "component__tooltip-wrapper";

const TooltipWrapper = ({
  children,
  tipContent,
  position = "bottom",
  isDelayed,
  className,
}: ITooltipWrapperProps): JSX.Element => {
  const classname = classnames(baseClass, className);
  const tipClass = isDelayed
    ? `${baseClass}__tip-text delayed-tip`
    : `${baseClass}__tip-text`;

  return (
    <div className={classname} data-position={position}>
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

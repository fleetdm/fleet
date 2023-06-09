import classnames from "classnames";
import React from "react";

import * as DOMPurify from "dompurify";

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

  const sanitizedTipContent = DOMPurify.sanitize(tipContent);

  return (
    <div className={classname} data-position={position}>
      <div className={`${baseClass}__element`}>
        {children}
        <div className={`${baseClass}__underline`} data-text={children} />
      </div>
      <div
        className={tipClass}
        dangerouslySetInnerHTML={{ __html: sanitizedTipContent }}
      />
    </div>
  );
};

export default TooltipWrapper;

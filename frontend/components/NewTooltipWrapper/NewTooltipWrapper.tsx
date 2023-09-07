import classnames from "classnames";
import React from "react";
import { Tooltip as ReactTooltip5 } from "react-tooltip-5";
import ReactTooltip from "react-tooltip";

import { uniqueId } from "lodash";
import { COLORS } from "styles/var/colors";

interface INewTooltipWrapperProps {
  children: string;
  tipContent: string;
  position?: "top" | "bottom";
  isDelayed?: boolean;
  underline?: boolean;
  // wrapperCustomClass?: string;
  className?: string;
  elementCustomClass?: string;
  // tipCustomClass?: string;
  tooltipClass?: string;
}

const baseClass = "component__tooltip-wrapper";

const NewTooltipWrapper = ({
  children,
  tipContent,
  position = "bottom",
  isDelayed,
  underline = true,
  // wrapperCustomClass,
  className,
  elementCustomClass,
  // tipCustomClass,
  tooltipClass, // to work with current usage, using above tipCustomClass would be more clear
}: // positionOverrides = { leftAdj: 54, topAdj: -3 },
INewTooltipWrapperProps): JSX.Element => {
  const wrapperClassNames = classnames(baseClass, className, {
    // [`${baseClass}__${wrapperCustomClass}`]: !!wrapperCustomClass,
  });

  const elementClassNames = classnames(`${baseClass}__element`, {
    [`${baseClass}__${elementCustomClass}`]: !!elementCustomClass,
    [`${baseClass}__underline`]: underline,
  });

  const tipClassNames = classnames(`${baseClass}__tip-text`, tooltipClass, {
    // [`${baseClass}__${tipCustomClass}`]: !!tipCustomClass,
    [`${baseClass}__tip-text__top`]: position === "top",
    [`${baseClass}__tip-text__bottom`]: position === "bottom",
  });
  const tipId = uniqueId();

  return (
    <span className={wrapperClassNames}>
      <div className={elementClassNames} data-tip data-for={tipId}>
        {children}
      </div>
      <ReactTooltip
        className={tipClassNames}
        type="dark"
        effect="solid"
        id={tipId}
        delayShow={isDelayed ? 500 : undefined}
        delayHide={isDelayed ? 500 : undefined}
        backgroundColor={COLORS["tooltip-bg"]}
      >
        {tipContent}
      </ReactTooltip>
    </span>
  );
  // return (
  //   <span className={wrapperClasses}>
  //     <div className={elementClasses} data-tooltip-id={tipId}>
  //       {children}
  //       {/* {underline && (
  //         <span className={`${baseClass}__underline`} data-text={children} />
  //       )} */}
  //     </div>
  //     <ReactTooltip5
  //       // className={`${baseClass}__tip-text`}
  //       place={position}
  //       variant="dark"
  //       // float="solid"
  //       id={tipId}
  //       delayHide={delayHide}
  //     >
  //       {tipContent}
  //     </ReactTooltip5>
  //   </span>
  // );
};

export default NewTooltipWrapper;

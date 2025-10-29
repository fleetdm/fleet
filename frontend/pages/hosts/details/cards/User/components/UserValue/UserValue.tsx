import React, { useRef } from "react";

import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "user-value";

interface IUserValueProps {
  values: string[];
}

const UserValue = ({ values }: IUserValueProps) => {
  const displayNameRef = useRef<HTMLSpanElement>(null);
  const isTruncated = useCheckTruncatedElement(displayNameRef);

  let content: React.ReactNode = null;

  if (values.length === 0) {
    // no values content
    content = DEFAULT_EMPTY_CELL_VALUE;
  } else if (values.length === 1) {
    // single value content
    content = (
      <TooltipWrapper
        tipContent={values[0]}
        underline={false}
        disableTooltip={!isTruncated}
        position="top"
        showArrow
      >
        <span ref={displayNameRef} className={`${baseClass}__single`}>
          {values[0]}
        </span>
      </TooltipWrapper>
    );
  } else {
    // multiple values content. dont include the first value in the tooltip.
    const tipContent = values.slice(1).map((value) => (
      <div key={value} className={`${baseClass}__tooltip-item`}>
        {value}
      </div>
    ));
    content = (
      <>
        <TooltipWrapper
          tipContent={values[0]}
          underline={false}
          disableTooltip={!isTruncated}
          position="top"
          showArrow
        >
          <span ref={displayNameRef} className={`${baseClass}__multi`}>
            {values[0]}
          </span>
        </TooltipWrapper>
        {/* conditionally render the space character as it only is needed when the
        value text is not truncated */}
        {!isTruncated ? " " : null}
        <TooltipWrapper tipContent={tipContent} position="bottom-start">
          <span>+ {values.length - 1} more</span>
        </TooltipWrapper>
      </>
    );
  }

  return <div className={`${baseClass}`}>{content}</div>;
};

export default UserValue;

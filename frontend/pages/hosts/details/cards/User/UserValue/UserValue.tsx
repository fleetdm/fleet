import React, { useRef } from "react";

import { useTruncatedElement } from "utilities/dom_utils";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "user-value";

interface IUserValueProps {
  values: string[];
}

const UserValue = ({ values }: IUserValueProps) => {
  const displayNameRef = useRef<HTMLSpanElement>(null);
  const isTruncated = useTruncatedElement(displayNameRef);

  let content: React.ReactNode = null;

  if (values.length === 0) {
    // no values content
    content = DEFAULT_EMPTY_CELL_VALUE;
  } else if (values.length === 1) {
    // single value content
    content = (
      <TooltipWrapper
        tipContent={values[0]}
        position="bottom"
        underline={false}
        disableTooltip={!isTruncated}
        showArrow
      >
        <span ref={displayNameRef} className={`${baseClass}__single`}>
          {values[0]}
        </span>
      </TooltipWrapper>
    );
  } else {
    // multiple values content
    const tipContent = values.map((value) => (
      <div key={value} className={`${baseClass}__tooltip-item`}>
        {value}
      </div>
    ));
    content = (
      <>
        <TooltipWrapper
          tipContent={values[0]}
          position="bottom"
          underline={false}
          disableTooltip={!isTruncated}
          showArrow
        >
          <span ref={displayNameRef} className={`${baseClass}__multi`}>
            {values[0]}
          </span>
        </TooltipWrapper>{" "}
        <TooltipWrapper tipContent={tipContent} position="bottom" showArrow>
          <span>+ {values.length - 1} more</span>
        </TooltipWrapper>
      </>
    );
  }

  return <div className={`${baseClass}`}>{content}</div>;
};

export default UserValue;

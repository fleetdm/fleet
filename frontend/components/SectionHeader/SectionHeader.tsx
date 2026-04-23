import React from "react";
import classnames from "classnames";

import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "section-header";

interface ISectionHeaderProps {
  title: string;
  subTitle?: React.ReactNode;
  details?: JSX.Element;
  wrapperCustomClass?: string;
  alignLeftHeaderVertically?: boolean;
  greySubtitle?: boolean;
  /** When provided, the section title becomes hoverable and shows this
   * content in a tooltip. */
  titleTooltipContent?: React.ReactNode;
}

const SectionHeader = ({
  title,
  subTitle,
  details,
  wrapperCustomClass,
  alignLeftHeaderVertically,
  greySubtitle,
  titleTooltipContent,
}: ISectionHeaderProps) => {
  const wrapperClassnames = classnames(baseClass, wrapperCustomClass);
  const leftHeaderClassnames = classnames(`${baseClass}__left-header`, {
    [`${baseClass}__left-header--vertical`]: alignLeftHeaderVertically,
  });
  const subTitleClassnames = classnames(`${baseClass}__sub-title`, {
    [`${baseClass}__sub-title--grey`]: greySubtitle,
  });

  const titleNode = titleTooltipContent ? (
    <TooltipWrapper tipContent={titleTooltipContent} position="top">
      {title}
    </TooltipWrapper>
  ) : (
    title
  );

  return (
    <div className={wrapperClassnames}>
      <div className={leftHeaderClassnames}>
        <h2>{titleNode}</h2>
        {subTitle && <div className={subTitleClassnames}>{subTitle}</div>}
      </div>
      {details && <div className={`${baseClass}__right-header`}>{details}</div>}
    </div>
  );
};

export default SectionHeader;

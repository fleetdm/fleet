import React from "react";
import classnames from "classnames";

const baseClass = "section-header";

interface ISectionHeaderProps {
  title: string;
  subTitle?: React.ReactNode;
  details?: JSX.Element;
  wrapperCustomClass?: string;
  alignLeftHeaderVertically?: boolean;
  greySubtitle?: boolean;
}

const SectionHeader = ({
  title,
  subTitle,
  details,
  wrapperCustomClass,
  alignLeftHeaderVertically,
  greySubtitle,
}: ISectionHeaderProps) => {
  const wrapperClassnames = classnames(baseClass, wrapperCustomClass);
  const leftHeaderClassnames = classnames(`${baseClass}__left-header`, {
    [`${baseClass}__left-header--vertical`]: alignLeftHeaderVertically,
  });
  const subTitleClassnames = classnames(`${baseClass}__sub-title`, {
    [`${baseClass}__sub-title--grey`]: greySubtitle,
  });

  return (
    <div className={wrapperClassnames}>
      <div className={leftHeaderClassnames}>
        <h2>{title}</h2>
        {subTitle && <div className={subTitleClassnames}>{subTitle}</div>}
      </div>
      {details && <div className={`${baseClass}__right-header`}>{details}</div>}
    </div>
  );
};

export default SectionHeader;

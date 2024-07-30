import React from "react";
import classnames from "classnames";

const baseClass = "section-header";

interface ISectionHeaderProps {
  title: string;
  subTitle?: React.ReactNode;
  details?: JSX.Element;
  className?: string;
}

const SectionHeader = ({
  title,
  subTitle,
  details,
  className,
}: ISectionHeaderProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      <div className={`${baseClass}__left-header`}>
        <h2>{title}</h2>
        {subTitle && (
          <div className={`${baseClass}__sub-title`}>{subTitle}</div>
        )}
      </div>
      {details && <div className={`${baseClass}__right-header`}>{details}</div>}
    </div>
  );
};

export default SectionHeader;

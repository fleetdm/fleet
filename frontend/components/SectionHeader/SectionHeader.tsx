import React from "react";
import classnames from "classnames";

const baseClass = "section-header";

interface ISectionHeaderProps {
  title: string;
  subTitle?: React.ReactNode;
  className?: string;
}

const SectionHeader = ({ title, subTitle, className }: ISectionHeaderProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      <h2>{title}</h2>
      {subTitle && <div className={`${baseClass}__sub-title`}>{subTitle}</div>}
    </div>
  );
};

export default SectionHeader;

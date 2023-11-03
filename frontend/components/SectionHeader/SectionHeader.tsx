import React from "react";

const baseClass = "section-header";

interface ISectionHeaderProps {
  title: string;
  subTitle?: React.ReactNode;
}

const SectionHeader = ({ title, subTitle }: ISectionHeaderProps) => {
  return (
    <div className={baseClass}>
      <h2>{title}</h2>
      {subTitle && <div className={`${baseClass}__sub-title`}>{subTitle}</div>}
    </div>
  );
};

export default SectionHeader;

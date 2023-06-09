import React from "react";

const baseClass = "section-header";

interface ISectionHeaderProps {
  title: string;
}

const SectionHeader = ({ title }: ISectionHeaderProps) => {
  return <h2 className={baseClass}>{title}</h2>;
};

export default SectionHeader;

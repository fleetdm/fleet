import React from "react";
import classnames from "classnames";

import SectionHeader from "components/SectionHeader";

const baseClass = "settings-section";

interface ISettingsSectionProps {
  title: string;
  children: React.ReactNode;
  className?: string;
}

/** This component encapsulates the common styles for each settings section */
const SettingsSection = ({
  title,
  children,
  className,
}: ISettingsSectionProps) => {
  const classes = classnames(baseClass, className);

  return (
    <section className={classes}>
      <SectionHeader title={title} wrapperCustomClass={`${baseClass}__title`} />
      <>{children}</>
    </section>
  );
};

export default SettingsSection;

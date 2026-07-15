import React from "react";
import classnames from "classnames";

import SectionHeader from "components/SectionHeader";

const baseClass = "settings-section";

interface ISettingsSectionProps {
  title: string;
  subTitle?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  id?: string;
}

/** This component encapsulates the common styles for each settings section */
const SettingsSection = ({
  title,
  subTitle,
  children,
  className,
  id,
}: ISettingsSectionProps) => {
  const classes = classnames(baseClass, className);

  // TODO: For now we assume any subTitle passed in equals grey text and vertical alignment, if the use case change we need to expose these variables to the caller.
  return (
    <section className={classes} id={id}>
      <SectionHeader
        title={title}
        subTitle={subTitle}
        greySubtitle={!!subTitle}
        alignLeftHeaderVertically={!!subTitle}
        wrapperCustomClass={`${baseClass}__title`}
      />
      <>{children}</>
    </section>
  );
};

export default SettingsSection;

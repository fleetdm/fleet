import React from "react";

import Card from "components/Card";
import Icon from "components/Icon";
import { IconNames } from "components/icons";

const baseClass = "section-card";

interface ISectionCardProps {
  children: React.ReactNode;
  header?: string;
  iconName?: IconNames;
  cta?: JSX.Element;
  // className?: string; TODO: If we want custom classNames
}

const SectionCard = ({
  children,
  header,
  iconName,
  cta,
}: ISectionCardProps) => {
  return (
    <Card className={baseClass} color="gray">
      <div className={`${baseClass}__content-wrapper`}>
        {iconName && <Icon name={iconName} />}
        <div className={`${baseClass}__content`}>
          {header && <h3>{header}</h3>}
          {children}
        </div>
      </div>
      {cta && <div className={`${baseClass}__cta`}>{cta}</div>}
    </Card>
  );
};

export default SectionCard;

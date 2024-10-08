import React from "react";

import Card from "components/Card";
import Icon from "components/Icon";
import { IconNames } from "components/icons";
import classnames from "classnames";

const baseClass = "section-card";

interface ISectionCardProps {
  children: React.ReactNode;
  header?: string;
  iconName?: IconNames;
  cta?: JSX.Element;
  className?: string;
}

const SectionCard = ({
  children,
  header,
  iconName,
  cta,
  className,
}: ISectionCardProps) => {
  const cardClasses = classnames(baseClass, className);

  return (
    <Card className={cardClasses} color="gray">
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

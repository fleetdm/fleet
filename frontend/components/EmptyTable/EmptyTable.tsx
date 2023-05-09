import React from "react";
import classnames from "classnames";
import Icon from "components/Icon";
import { IconNames } from "components/icons";

const baseClass = "empty-table";

export interface IEmptyTableProps {
  header?: JSX.Element | string;
  info?: JSX.Element | string;
  additionalInfo?: JSX.Element | string;
  iconName?: IconNames;
  primaryButton?: JSX.Element;
  secondaryButton?: JSX.Element;
  className?: string;
}

const EmptyTable = ({
  iconName,
  header,
  info,
  additionalInfo,
  className,
  primaryButton,
  secondaryButton,
}: IEmptyTableProps): JSX.Element => {
  const emptyTableClass = classnames(`${baseClass}__container`, className);

  return (
    <div className={emptyTableClass}>
      {iconName && (
        <div className={`${baseClass}__image-wrapper`}>
          <Icon name={iconName} />
        </div>
      )}
      <div className={`${baseClass}__inner`}>
        {header && <h3>{header}</h3>}
        {info && <p>{info}</p>}
        {additionalInfo && <p>{additionalInfo}</p>}
      </div>
      {primaryButton && (
        <div className={`${baseClass}__cta-buttons`}>
          {primaryButton}
          {secondaryButton && secondaryButton}
        </div>
      )}
    </div>
  );
};

export default EmptyTable;

import React from "react";
import { IconNames } from "components/icons";
import classnames from "classnames";
import Icon from "components/Icon";

const baseClass = "empty-table";

export interface IEmptyTableProps {
  iconName?: IconNames;
  header?: JSX.Element | string;
  info?: JSX.Element | string;
  additionalInfo?: JSX.Element | string;
  className?: string;
  primaryButton?: JSX.Element;
  secondaryButton?: JSX.Element;
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
  const emptyTableClassname = classnames(`${baseClass}__container`, className);

  return (
    <div className={emptyTableClassname}>
      {!!iconName && (
        <div className={`${baseClass}__image-wrapper`}>
          <Icon name={iconName} />
        </div>
      )}
      <div className={`${baseClass}__inner`}>
        {header && <h2>{header}</h2>}
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

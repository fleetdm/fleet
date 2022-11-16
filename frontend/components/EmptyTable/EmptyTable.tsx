// This component is used on the ManageSoftwarePage.tsx

import React from "react";
import { IconNames } from "components/icons";
import classnames from "classnames";
import Icon from "components/Icon";

const baseClass = "empty-table";

export interface IEmptyTableProps {
  iconName?: IconNames;
  headerText?: string;
  infoText?: JSX.Element | string;
  className?: string;
  primaryButton?: JSX.Element;
  secondaryButton?: JSX.Element;
}

const EmptyTable = ({
  iconName,
  headerText,
  infoText,
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
        {headerText && <h2>{headerText}</h2>}
        {infoText && <p>{infoText}</p>}
      </div>
      {!!primaryButton && (
        <div className={`${baseClass}__cta-buttons`}>
          {primaryButton}
          {secondaryButton && secondaryButton}
        </div>
      )}
    </div>
  );
};

export default EmptyTable;

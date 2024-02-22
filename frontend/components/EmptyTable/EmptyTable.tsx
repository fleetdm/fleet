import React from "react";
import classnames from "classnames";
import Graphic from "components/Graphic";
import { GraphicNames } from "components/graphics";

const baseClass = "empty-table";

export interface IEmptyTableProps {
  header?: JSX.Element | string;
  info?: JSX.Element | string;
  additionalInfo?: JSX.Element | string;
  graphicName?: GraphicNames;
  primaryButton?: JSX.Element;
  secondaryButton?: JSX.Element;
  className?: string;
}

const EmptyTable = ({
  graphicName,
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
      {graphicName && (
        <div className={`${baseClass}__image-wrapper`}>
          <Graphic name={graphicName} />
        </div>
      )}
      <div className={`${baseClass}__inner`}>
        {header && <h3>{header}</h3>}
        {info && <div className={`${baseClass}__info`}>{info}</div>}
        {additionalInfo && (
          <div className={`${baseClass}__additional-info`}>
            {additionalInfo}
          </div>
        )}
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

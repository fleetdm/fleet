import React from "react";
import classnames from "classnames";

import Graphic from "components/Graphic";
import { GraphicNames } from "components/graphics";

const baseClass = "empty-state";

// Number of visible ghost rows (header + data rows that fill the container)
const GHOST_ROW_COUNT = 8;

export interface IEmptyStateProps {
  header?: JSX.Element | string;
  info?: JSX.Element | string;
  additionalInfo?: JSX.Element | string;
  graphicName?: GraphicNames;
  primaryButton?: JSX.Element;
  secondaryButton?: JSX.Element;
  /** "default" renders 3 ghost columns (page-level), "small" renders 2 (modal-level) */
  width?: "default" | "small";
  className?: string;
}

/** A single ghost column with a header row and skeleton data rows. */
const GhostColumn = ({
  skeletonWidth,
  isFirst,
}: {
  skeletonWidth: number;
  isFirst?: boolean;
}) => (
  <div
    className={classnames(`${baseClass}__ghost-col`, {
      [`${baseClass}__ghost-col--first`]: isFirst,
    })}
  >
    <div className={`${baseClass}__ghost-header`}>
      <div
        className={`${baseClass}__ghost-skeleton`}
        style={{ width: 80 }}
      />
    </div>
    {Array.from({ length: GHOST_ROW_COUNT }, (_, i) => (
      <div key={i} className={`${baseClass}__ghost-cell`}>
        <div
          className={`${baseClass}__ghost-skeleton`}
          style={{ width: skeletonWidth }}
        />
      </div>
    ))}
  </div>
);

const EmptyState = ({
  header,
  info,
  additionalInfo,
  graphicName,
  primaryButton,
  secondaryButton,
  width = "default",
  className,
}: IEmptyStateProps): JSX.Element => {
  const containerClass = classnames(baseClass, className, {
    [`${baseClass}--small`]: width === "small",
  });

  return (
    <div className={containerClass}>
      <div className={`${baseClass}__ghost-table`} aria-hidden="true">
        <GhostColumn skeletonWidth={280} isFirst />
        <GhostColumn skeletonWidth={120} />
        {width === "default" && <GhostColumn skeletonWidth={60} />}
      </div>
      <div className={`${baseClass}__content-wrapper`}>
        <div className={`${baseClass}__content`}>
          {graphicName && (
            <div className={`${baseClass}__image-wrapper`}>
              <Graphic name={graphicName} />
            </div>
          )}
          {header && <h3>{header}</h3>}
          {info && <div className={`${baseClass}__info`}>{info}</div>}
          {additionalInfo && (
            <div className={`${baseClass}__additional-info`}>
              {additionalInfo}
            </div>
          )}
          {(primaryButton || secondaryButton) && (
            <div className={`${baseClass}__cta-buttons`}>
              {primaryButton}
              {secondaryButton}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default EmptyState;

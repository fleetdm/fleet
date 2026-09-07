import React from "react";
import classnames from "classnames";

const baseClass = "empty-state";

// Number of visible ghost rows (header + data rows that fill the container)
const GHOST_ROW_COUNT = 8;
const GHOST_FORM_FIELD_COUNT = 3;

/** Deterministic per-row width jitter so skeletons don't align vertically. */
const varyWidth = (base: number, index: number) => {
  const offsets = [0, 14, -8, 22, -4, 18, -12, 26, 6, -10];
  return base + offsets[index % offsets.length];
};

export interface IEmptyStateProps {
  header?: JSX.Element | string;
  info?: JSX.Element | string;
  additionalInfo?: JSX.Element | string;
  primaryButton?: JSX.Element;
  secondaryButton?: JSX.Element;
  /** "default" renders 3 ghost columns (page-level), "small" renders 2 (modal-level) */
  width?: "default" | "small";
  /**
   * "list" renders 1 ghost column with no header row.
   * "header-list" renders 1 ghost column with a header row.
   * "form" renders ghost form fields with a save button.
   * Defaults to undefined (standard multi-column table ghost).
   */
  variant?: "list" | "header-list" | "form";
  className?: string;
}

/** A single ghost column with an optional header row and skeleton data rows. */
const GhostColumn = ({
  skeletonWidth,
  isFirst,
  showHeader = true,
  twoLines = false,
}: {
  skeletonWidth: number;
  isFirst?: boolean;
  showHeader?: boolean;
  twoLines?: boolean;
}) => (
  <div
    className={classnames(`${baseClass}__ghost-col`, {
      [`${baseClass}__ghost-col--first`]: isFirst,
    })}
  >
    {showHeader && (
      <div className={`${baseClass}__ghost-header`}>
        <div className={`${baseClass}__ghost-skeleton`} style={{ width: 80 }} />
      </div>
    )}
    {Array.from({ length: GHOST_ROW_COUNT }, (_, i) => (
      <div key={i} className={`${baseClass}__ghost-cell`}>
        <div
          className={`${baseClass}__ghost-skeleton`}
          style={{ width: varyWidth(skeletonWidth, i) }}
        />
        {twoLines && (
          <div
            className={`${baseClass}__ghost-skeleton ${baseClass}__ghost-skeleton--secondary`}
            style={{ width: varyWidth(skeletonWidth * 0.6, i + 3) }}
          />
        )}
      </div>
    ))}
  </div>
);

/** Ghost form with label+input field pairs and a button skeleton. */
const GhostForm = () => (
  <div className={`${baseClass}__ghost-form`}>
    {Array.from({ length: GHOST_FORM_FIELD_COUNT }, (_, i) => (
      <div key={i} className={`${baseClass}__ghost-field`}>
        <div
          className={`${baseClass}__ghost-skeleton ${baseClass}__ghost-label`}
          style={{ width: varyWidth(80, i) }}
        />
        <div
          className={`${baseClass}__ghost-skeleton ${baseClass}__ghost-input`}
        />
      </div>
    ))}
    <div
      className={`${baseClass}__ghost-skeleton ${baseClass}__ghost-button`}
    />
  </div>
);

const EmptyState = ({
  header,
  info,
  additionalInfo,
  primaryButton,
  secondaryButton,
  width = "default",
  variant,
  className,
}: IEmptyStateProps): JSX.Element => {
  const isList = variant === "list" || variant === "header-list";
  const isForm = variant === "form";
  const showGhostHeader = variant !== "list" && variant !== "form";

  const containerClass = classnames(baseClass, className, {
    [`${baseClass}--small`]: width === "small",
    [`${baseClass}--list`]: isList,
    [`${baseClass}--form`]: isForm,
  });

  const renderGhostTable = () => {
    if (isForm) {
      return <GhostForm />;
    }
    if (isList) {
      return (
        <GhostColumn
          skeletonWidth={280}
          isFirst
          showHeader={showGhostHeader}
          twoLines
        />
      );
    }
    return (
      <>
        <GhostColumn skeletonWidth={280} isFirst />
        <GhostColumn skeletonWidth={120} />
        {width === "default" && <GhostColumn skeletonWidth={60} />}
      </>
    );
  };

  return (
    <div className={containerClass}>
      <div className={`${baseClass}__ghost-table`} aria-hidden="true">
        {renderGhostTable()}
      </div>
      <div className={`${baseClass}__content-wrapper`}>
        <div className={`${baseClass}__content`}>
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

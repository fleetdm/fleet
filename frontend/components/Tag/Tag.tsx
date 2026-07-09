import React from "react";
import classnames from "classnames";

import Icon from "components/Icon";

const baseClass = "tag";

export type TagType = "static" | "clickable" | "dismissible";

interface ITagProps {
  type?: TagType;
  children: React.ReactNode;
  /** Required when `type` is `"clickable"` */
  onClick?: () => void;
  /** Required when `type` is `"dismissible"` */
  onDismiss?: () => void;
  /** Accessible name (and native tooltip) for the dismiss button */
  dismissLabel?: string;
  disabled?: boolean;
  className?: string;
}

const Tag = ({
  type = "static",
  children,
  onClick,
  onDismiss,
  dismissLabel = "Remove",
  disabled,
  className,
}: ITagProps) => {
  const classNames = classnames(baseClass, className, {
    [`${baseClass}--clickable`]: type === "clickable",
    [`${baseClass}--dismissible`]: type === "dismissible",
  });

  if (type === "clickable") {
    return (
      <button
        type="button"
        className={classNames}
        disabled={disabled}
        onClick={onClick}
      >
        {children}
      </button>
    );
  }

  if (type === "dismissible") {
    return (
      <div className={classNames}>
        <span className={`${baseClass}__label`}>{children}</span>
        <button
          type="button"
          className={`${baseClass}__dismiss`}
          disabled={disabled}
          onClick={onDismiss}
          aria-label={dismissLabel}
          title={dismissLabel}
        >
          <Icon name="close" color="core-fleet-black" size="small" />
        </button>
      </div>
    );
  }

  return <div className={classNames}>{children}</div>;
};

export default Tag;

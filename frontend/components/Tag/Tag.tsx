import React from "react";
import classnames from "classnames";

import Icon from "components/Icon";

const baseClass = "tag";

export type TagType = "static" | "clickable" | "dismissible";

interface ITagBaseProps {
  children: React.ReactNode;
  disabled?: boolean;
  className?: string;
}

interface IStaticTagProps extends ITagBaseProps {
  type?: "static";
  onClick?: never;
  onDismiss?: never;
  dismissLabel?: never;
}

interface IClickableTagProps extends ITagBaseProps {
  type: "clickable";
  onClick: () => void;
  onDismiss?: never;
  dismissLabel?: never;
}

interface IDismissibleTagProps extends ITagBaseProps {
  type: "dismissible";
  onClick?: never;
  onDismiss: () => void;
  /** Accessible name (and native tooltip) for the dismiss button. Defaults to "Dismiss". */
  dismissLabel?: string;
}

type ITagProps = IStaticTagProps | IClickableTagProps | IDismissibleTagProps;

const Tag = (props: ITagProps) => {
  const { children, disabled, className } = props;

  const classNames = classnames(baseClass, className, {
    [`${baseClass}--clickable`]: props.type === "clickable",
    [`${baseClass}--dismissible`]: props.type === "dismissible",
  });

  if (props.type === "clickable") {
    return (
      <button
        type="button"
        className={classNames}
        disabled={disabled}
        onClick={props.onClick}
      >
        {children}
      </button>
    );
  }

  if (props.type === "dismissible") {
    const dismissLabel = props.dismissLabel ?? "Dismiss";

    return (
      <span className={classNames}>
        <span className={`${baseClass}__label`}>{children}</span>
        <button
          type="button"
          className={`${baseClass}__dismiss`}
          disabled={disabled}
          onClick={props.onDismiss}
          aria-label={dismissLabel}
          title={dismissLabel}
        >
          <Icon name="close" color="core-fleet-black" size="small" />
        </button>
      </span>
    );
  }

  return <span className={classNames}>{children}</span>;
};

export default Tag;

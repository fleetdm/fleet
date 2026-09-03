import React from "react";
import classnames from "classnames";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "tag";

interface ITagBaseProps {
  children: React.ReactNode;
  /** Default: "large" (28px). Per design, use "small" (24px) sparingly and
   * "xsmall" (20px) only inline with table cell text. */
  size?: "large" | "small" | "xsmall";
  className?: string;
  /** Wraps the tag in a tooltip that shows this content on hover */
  tooltip?: JSX.Element | string;
}

interface IStaticTagProps extends ITagBaseProps {
  type?: "static";
  onClick?: never;
  onDismiss?: never;
  dismissLabel?: never;
  /** Static tags are non-interactive — disabled doesn't apply. */
  disabled?: never;
}

interface IClickableTagProps extends ITagBaseProps {
  type: "clickable";
  onClick: () => void;
  onDismiss?: never;
  dismissLabel?: never;
  disabled?: boolean;
}

interface IDismissibleTagProps extends ITagBaseProps {
  type: "dismissible";
  onClick?: never;
  onDismiss: () => void;
  /** Accessible name for the dismiss button (screen readers only, no native tooltip). Defaults to "Dismiss". */
  dismissLabel?: string;
  /** Dismissible tags are always interactive — no production caller disables them. */
  disabled?: never;
}

type ITagProps = IStaticTagProps | IClickableTagProps | IDismissibleTagProps;

const Tag = (props: ITagProps) => {
  const { children, className, tooltip } = props;

  const classNames = classnames(baseClass, className, {
    [`${baseClass}--clickable`]: props.type === "clickable",
    [`${baseClass}--dismissible`]: props.type === "dismissible",
    [`${baseClass}--small`]: props.size === "small",
    [`${baseClass}--xsmall`]: props.size === "xsmall",
  });

  let content: JSX.Element;

  if (props.type === "clickable") {
    content = (
      <button
        type="button"
        className={classNames}
        disabled={props.disabled}
        onClick={props.onClick}
      >
        {children}
      </button>
    );
  } else if (props.type === "dismissible") {
    const dismissLabel = props.dismissLabel ?? "Dismiss";

    content = (
      <span className={classNames}>
        <span className={`${baseClass}__label`}>{children}</span>
        <button
          type="button"
          className={`${baseClass}__dismiss`}
          onClick={props.onDismiss}
          aria-label={dismissLabel}
        >
          <Icon name="close" color="core-fleet-black" size="small" />
        </button>
      </span>
    );
  } else {
    content = <span className={classNames}>{children}</span>;
  }

  if (!tooltip) {
    return content;
  }

  return (
    <TooltipWrapper
      tipContent={tooltip}
      showArrow
      underline={false}
      position="top"
      tipOffset={12}
      delayShow={300}
      fixedPositionStrategy
    >
      {content}
    </TooltipWrapper>
  );
};

export default Tag;

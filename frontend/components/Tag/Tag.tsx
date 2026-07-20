import React from "react";
import classnames from "classnames";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "tag";

export type TagType = "static" | "clickable" | "dismissible";

interface ITagBaseProps {
  children: React.ReactNode;
  /** Default: "large" (28px). Per design, use "small" (24px) sparingly. */
  size?: "large" | "small";
  disabled?: boolean;
  className?: string;
  /** Wraps the tag in a tooltip that shows this content on hover */
  tooltip?: JSX.Element | string;
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
  const { children, disabled, className, tooltip } = props;

  const classNames = classnames(baseClass, className, {
    [`${baseClass}--clickable`]: props.type === "clickable",
    [`${baseClass}--dismissible`]: props.type === "dismissible",
    [`${baseClass}--small`]: props.size === "small",
  });

  let content: JSX.Element;

  if (props.type === "clickable") {
    content = (
      <button
        type="button"
        className={classNames}
        disabled={disabled}
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
          disabled={disabled}
          onClick={props.onDismiss}
          aria-label={dismissLabel}
          title={dismissLabel}
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
      delayInMs={300}
    >
      {content}
    </TooltipWrapper>
  );
};

export default Tag;

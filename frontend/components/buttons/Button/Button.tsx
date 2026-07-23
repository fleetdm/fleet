import React from "react";
import classnames from "classnames";
import Spinner from "components/Spinner";
import Icon from "components/Icon";

const baseClass = "button";

export type ButtonVariant =
  | "default"
  | "alert"
  | "pill"
  | "link" // Looks like CustomLink with animated underline on hover
  | "secondary" // Bordered secondary button (off-white fill + border). The new preferred secondary — see #35329.
  | "subdued" // Low-emphasis borderless text + icon button. Not to be confused with a link.
  | "unstyled" // Avoid as much as possible (used in registration breadcrumbs, 404/500, an old button dropdown)
  | "unstyled-modal-query"
  | "oversized";

export interface IButtonProps {
  autofocus?: boolean;
  children: React.ReactNode;
  className?: string;
  disabled?: boolean;
  tabIndex?: number;
  type?: "button" | "submit" | "reset";
  /** Text shown on tooltip when hovering over a button */
  title?: string;
  /** Default: "default" */
  variant?: ButtonVariant;
  onClick?:
    | ((value?: any) => void)
    | ((
        evt:
          | React.MouseEvent<HTMLButtonElement>
          | React.KeyboardEvent<HTMLButtonElement>
      ) => void);
  isLoading?: boolean;
  customOnKeyDown?: (e: React.KeyboardEvent) => void;
  ariaHasPopup?:
    | boolean
    | "false"
    | "true"
    | "menu"
    | "listbox"
    | "tree"
    | "grid"
    | "dialog";
  ariaExpanded?: boolean;
  ariaLabel?: string;
  ariaPressed?: boolean;
  /** Small: 1/2 the padding, Wide: 200px */
  size?: "small" | "wide" | "default";
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface IButtonState {}

interface Inputs {
  button?: HTMLButtonElement;
}

class Button extends React.Component<IButtonProps, IButtonState> {
  static defaultProps = {
    type: "button",
    variant: "default",
  };

  componentDidMount(): void {
    const { autofocus } = this.props;
    const {
      inputs: { button },
    } = this;

    if (autofocus && button) {
      button.focus();
    }
  }

  setRef = (button: HTMLButtonElement): boolean => {
    this.inputs.button = button;

    return false;
  };

  inputs: Inputs = {};

  handleClick = (evt: React.MouseEvent<HTMLButtonElement>): void => {
    const { disabled, onClick } = this.props;

    if (disabled) {
      return;
    }

    if (onClick) {
      onClick(evt);
    }
  };

  handleKeyDown = (evt: React.KeyboardEvent<HTMLButtonElement>): void => {
    const { disabled, onClick } = this.props;

    if (disabled || evt.key !== "Enter") {
      return;
    }

    if (onClick) {
      onClick(evt as any);
    }
  };

  render(): JSX.Element {
    const { handleClick, handleKeyDown, setRef } = this;
    const {
      children,
      className,
      disabled,
      tabIndex,
      type,
      title,
      variant,
      isLoading,
      customOnKeyDown,
      ariaHasPopup,
      ariaExpanded,
      ariaLabel,
      ariaPressed,
      size,
    } = this.props;
    // The bordered "secondary" and borderless "subdued" variants render as a
    // square when their only content is an icon (no text label) — see #35329.
    // toArray strips false/null and flattens fragments, so we reliably detect a
    // lone <Icon> child while ignoring conditional or wrapped text content.
    const childArray = React.Children.toArray(children);
    const isIconOnly =
      (variant === "secondary" || variant === "subdued") &&
      childArray.length === 1 &&
      React.isValidElement(childArray[0]) &&
      childArray[0].type === Icon;
    const fullClassName = classnames(
      baseClass,
      `${baseClass}--${variant}`,
      className,
      {
        [`${baseClass}--${variant}__small`]: size === "small",
        [`${baseClass}__wide`]: size === "wide",
        [`${baseClass}--disabled`]: disabled,
        [`${baseClass}--icon-only`]: isIconOnly,
      }
    );
    const onWhite =
      variant === "link" ||
      variant === "secondary" ||
      variant === "subdued" ||
      variant === "pill" ||

    return (
      <button
        className={fullClassName}
        disabled={disabled}
        onClick={handleClick}
        onKeyDown={customOnKeyDown || handleKeyDown}
        tabIndex={tabIndex}
        type={type}
        title={title}
        ref={setRef}
        aria-haspopup={ariaHasPopup}
        aria-expanded={ariaExpanded}
        aria-label={ariaLabel}
        aria-pressed={ariaPressed}
      >
        <div className={isLoading ? "transparent-text" : "children-wrapper"}>
          {children}
        </div>
        {isLoading && <Spinner small button white={!onWhite} delay={0} />}
      </button>
    );
  }
}

export default Button;

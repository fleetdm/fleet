import React from "react";
import classnames from "classnames";
import Spinner from "components/Spinner";
import Icon from "components/Icon";
import { IconNames } from "components/icons";

const baseClass = "button";

export type ButtonVariant =
  | "default"
  | "alert"
  | "pill"
  | "grey-pill"
  | "link" // Looks like CustomLink with animated underline on hover
  | "secondary" // Bordered secondary button (off-white fill + border). The new preferred secondary — see #35329.
  | "subdued" // Low-emphasis borderless text + icon button. Not to be confused with a link.
  | "unstyled" // Avoid as much as possible (used in registration breadcrumbs, 404/500, an old button dropdown)
  | "unstyled-modal-query"
  | "oversized";

// Variants whose text color is white — icons default to white on these.
const WHITE_TEXT_VARIANTS: readonly ButtonVariant[] = [
  "default",
  "alert",
  "oversized",
];

// Only these variants participate in icon / iconPosition / icon-only styling.
const ICON_ENABLED_VARIANTS: readonly ButtonVariant[] = [
  "default",
  "alert",
  "secondary",
  "subdued",
];

export interface IButtonProps {
  autofocus?: boolean;
  children?: React.ReactNode;
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
  /**
   * Icon rendered inside the button. Sized automatically (16px on default,
   * 12px on `size="small"`) and colored to match the variant's text.
   * With no `children`, renders an icon-only square (secondary/subdued only).
   */
  icon?: IconNames;
  /** Where to render `icon` relative to the label. Ignored when icon-only. */
  iconPosition?: "left" | "right";
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
    iconPosition: "left",
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
      icon,
      iconPosition,
    } = this.props;
    const hasLabel = React.Children.count(children) > 0;
    // Square, centered layout for a bare icon on secondary/subdued — see #35329.
    const isIconOnly =
      !!icon && !hasLabel && (variant === "secondary" || variant === "subdued");
    const hasIconWithLabel =
      !!icon && hasLabel && ICON_ENABLED_VARIANTS.includes(variant!);
    const fullClassName = classnames(
      baseClass,
      `${baseClass}--${variant}`,
      className,
      {
        [`${baseClass}--${variant}__small`]: size === "small",
        [`${baseClass}__wide`]: size === "wide",
        [`${baseClass}--disabled`]: disabled,
        [`${baseClass}--icon-only`]: isIconOnly,
        [`${baseClass}--with-icon`]: hasIconWithLabel,
      }
    );
    // Variants with white text (dark backgrounds) — icons and the loading
    // spinner both render white so they're visible.
    const hasWhiteText = WHITE_TEXT_VARIANTS.includes(variant!);
    // Icons: 16px on default-size buttons, 12px on small-size buttons.
    const iconSize = size === "small" ? "small" : "medium";
    const iconColor = hasWhiteText ? "core-fleet-white" : undefined;
    const iconElement = icon && (
      <Icon name={icon} size={iconSize} color={iconColor} />
    );

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
        <div
          className={classnames("children-wrapper", {
            "transparent-text": isLoading,
          })}
        >
          {iconElement && iconPosition === "left" && iconElement}
          {children}
          {iconElement && iconPosition === "right" && iconElement}
        </div>
        {isLoading && <Spinner small button white={hasWhiteText} delay={0} />}
      </button>
    );
  }
}

export default Button;

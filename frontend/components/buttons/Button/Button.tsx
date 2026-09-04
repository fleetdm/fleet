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
  | "link" // Looks like CustomLink with animated underline on hover
  | "secondary" // Bordered secondary button (off-white fill + border). The new preferred secondary — see #35329.
  | "subdued" // Low-emphasis borderless text + icon button. Not to be confused with a link.
  | "unstyled" // Avoid as much as possible (used in registration breadcrumbs, 404/500, an old button dropdown)
  | "unstyled-modal-query"
  | "oversized";

// Variants whose text color is white — icons and the loading spinner default
// to white on these. `oversized` is in the list for the spinner; it doesn't
// render icons (not in ICON_ENABLED_VARIANTS).
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
  /**
   * `id` of the `<form>` this button submits, for a `type="submit"` button that
   * renders outside that form (e.g. inside a `ModalFooter` sibling). Lets the
   * form keep its `onSubmit` handler instead of moving submission to `onClick`.
   */
  formId?: string;
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
  ariaControls?: string;
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
      formId,
      title,
      variant,
      isLoading,
      customOnKeyDown,
      ariaHasPopup,
      ariaExpanded,
      ariaControls,
      ariaLabel,
      ariaPressed,
      size,
      icon,
      iconPosition,
    } = this.props;
    // Square, centered layout for a bare icon on secondary/subdued — see #35329.
    // Detects both the modern `icon` prop and the legacy `<Icon>` child pattern
    // used by call sites that need a color/className override on the icon.
    // `toArray` (not `Children.count`) so `{cond && "text"}` with a false `cond`
    // reads as "no label" — otherwise the false child still counts as a label
    // and the visually-lone icon skips its square styling.
    const childArray = React.Children.toArray(children);
    const hasLabel = childArray.length > 0;
    const hasLoneIconChild =
      !icon &&
      childArray.length === 1 &&
      React.isValidElement(childArray[0]) &&
      childArray[0].type === Icon;
    const isIconOnly =
      (variant === "secondary" || variant === "subdued") &&
      ((!!icon && !hasLabel) || hasLoneIconChild);
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
    // Match the icon to the button text: white on dark-fill variants,
    // ui-fleet-black-75 on the light secondary/subdued variants.
    let iconColor: "core-fleet-white" | "ui-fleet-black-75" | undefined;
    if (hasWhiteText) {
      iconColor = "core-fleet-white";
    } else if (variant === "secondary" || variant === "subdued") {
      iconColor = "ui-fleet-black-75";
    }
    // Skip the icon on variants that don't support it — otherwise the SVG
    // renders unspaced against the label, since the `--with-icon` gap rule is
    // scoped to ICON_ENABLED_VARIANTS in _styles.scss.
    const iconElement = icon && ICON_ENABLED_VARIANTS.includes(variant!) && (
      <Icon name={icon} size={iconSize} color={iconColor} />
    );
    // Icon-only buttons need an explicit accessible name. Browsers fall back
    // to `title` when `aria-label` is missing, so either satisfies a screen
    // reader; warn (in dev) when neither is set.
    if (
      process.env.NODE_ENV !== "production" &&
      isIconOnly &&
      !ariaLabel &&
      !title
    ) {
      // eslint-disable-next-line no-console
      console.warn(
        `Icon-only Button (icon="${
          icon ?? "unknown"
        }") has no ariaLabel or title — ` +
          `screen readers will announce it as an unlabeled button.`
      );
    }

    return (
      <button
        className={fullClassName}
        disabled={disabled}
        onClick={handleClick}
        onKeyDown={customOnKeyDown || handleKeyDown}
        tabIndex={tabIndex}
        type={type}
        form={formId}
        title={title}
        ref={setRef}
        aria-haspopup={ariaHasPopup}
        aria-expanded={ariaExpanded}
        aria-controls={ariaControls}
        aria-label={ariaLabel}
        aria-pressed={ariaPressed}
      >
        <div
          className={classnames("children-wrapper", {
            "transparent-text": isLoading,
          })}
        >
          {iconPosition === "left" && iconElement}
          {children}
          {iconPosition === "right" && iconElement}
        </div>
        {isLoading && <Spinner small button white={hasWhiteText} delay={0} />}
      </button>
    );
  }
}

export default Button;

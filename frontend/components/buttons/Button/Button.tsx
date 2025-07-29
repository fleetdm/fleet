import React from "react";
import classnames from "classnames";
import Spinner from "components/Spinner";

const baseClass = "button";

export type ButtonVariant =
  | "default"
  | "success"
  | "alert"
  | "pill"
  | "text-link" // Underlines on hover
  | "text-link-dark" // underline on hover, dark text
  | "text-icon"
  | "icon" // Buttons without text
  | "inverse"
  | "inverse-alert"
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
  /** Required for buttons that contain SVG icons using`stroke` instead of`fill` for proper hover styling */
  iconStroke?: boolean;
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
      iconStroke,
    } = this.props;
    const fullClassName = classnames(
      baseClass,
      `${baseClass}--${variant}`,
      className,
      {
        [`${baseClass}--disabled`]: disabled,
        [`${baseClass}--icon-stroke`]: iconStroke,
      }
    );
    const onWhite =
      variant === "text-link" ||
      variant === "inverse" ||
      variant === "text-icon" ||
      variant === "pill";

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
      >
        <div className={isLoading ? "transparent-text" : "children-wrapper"}>
          {children}
        </div>
        {isLoading && <Spinner small button white={!onWhite} />}
      </button>
    );
  }
}

export default Button;

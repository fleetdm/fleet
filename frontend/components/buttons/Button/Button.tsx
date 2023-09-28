import React from "react";
import classnames from "classnames";
import Spinner from "components/Spinner";

const baseClass = "button";

export type ButtonVariant =
  | "brand"
  | "success"
  | "alert"
  | "blue-green"
  | "grey"
  | "warning"
  | "link"
  | "label"
  | "text-link"
  | "text-icon"
  | "icon" // Buttons without text
  | "small-icon" // Buttons without text
  | "inverse"
  | "inverse-alert"
  | "block"
  | "unstyled"
  | "unstyled-modal-query"
  | "contextual-nav-item"
  | "small-text-icon"
  | "oversized";

export interface IButtonProps {
  autofocus?: boolean;
  children: React.ReactNode;
  className?: string;
  disabled?: boolean;
  size?: string;
  tabIndex?: number;
  type?: "button" | "submit" | "reset";
  title?: string;
  variant?: ButtonVariant;
  onClick?:
    | ((value?: any) => void)
    | ((evt: React.MouseEvent<HTMLButtonElement>) => void);
  isLoading?: boolean;
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface IButtonState {}

interface Inputs {
  button?: HTMLButtonElement;
}

class Button extends React.Component<IButtonProps, IButtonState> {
  static defaultProps = {
    size: "",
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

  handleClick = (evt: React.MouseEvent<HTMLButtonElement>): boolean => {
    const { disabled, onClick } = this.props;

    if (disabled) {
      return false;
    }

    if (onClick) {
      onClick(evt);
    }

    return false;
  };

  render(): JSX.Element {
    const { handleClick, setRef } = this;
    const {
      children,
      className,
      disabled,
      size,
      tabIndex,
      type,
      title,
      variant,
      isLoading,
    } = this.props;
    const fullClassName = classnames(
      baseClass,
      `${baseClass}--${variant}`,
      className,
      {
        [`${baseClass}--disabled`]: disabled,
        [`${baseClass}--${size}`]: size !== undefined,
      }
    );
    const onWhite =
      variant === "text-link" ||
      variant === "inverse" ||
      variant === "text-icon" ||
      variant === "label";

    return (
      <button
        className={fullClassName}
        disabled={disabled}
        onClick={handleClick}
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

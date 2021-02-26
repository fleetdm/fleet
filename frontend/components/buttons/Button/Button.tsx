import React from 'react';
import classnames from 'classnames';

const baseClass = 'button';

interface IButtonProps {
  autofocus: boolean;
  block: boolean;
  children: React.ReactChild;
  className: string;
  disabled: boolean;
  onClick: (evt: React.MouseEvent<HTMLButtonElement>) => boolean;
  size: string;
  tabIndex: number;
  type: string;
  title: string;
  variant: string;
}

interface Inputs {
  button?: HTMLButtonElement;
}

class Button extends React.Component<IButtonProps, null> {
  static defaultProps = {
    block: false,
    size: '',
    type: 'button',
    variant: 'default',
  };

  componentDidMount(): void {
    const { autofocus } = this.props;
    const { inputs: { button } } = this;

    if (autofocus && button) {
      button.focus();
    }
  }

  setRef = (button: HTMLButtonElement): boolean => {
    this.inputs.button = button;

    return false;
  }

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
  }

  render(): JSX.Element {
    const { handleClick, setRef } = this;
    const { block, children, className, disabled, size, tabIndex, type, title, variant } = this.props;
    const fullClassName = classnames(baseClass, `${baseClass}--${variant}`, className, {
      [`${baseClass}--block`]: block,
      [`${baseClass}--disabled`]: disabled,
      [`${baseClass}--${size}`]: size !== undefined,
    });

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
        {children}
      </button>
    );
  }
}

export default Button;

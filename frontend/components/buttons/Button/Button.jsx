import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

const baseClass = 'button';

class Button extends Component {
  static propTypes = {
    className: PropTypes.string,
    disabled: PropTypes.bool,
    onClick: PropTypes.func,
    text: PropTypes.string,
    type: PropTypes.string,
    variant: PropTypes.string,
  };

  static defaultProps = {
    variant: 'default',
  };

  handleClick = (evt) => {
    const { disabled, onClick } = this.props;

    if (disabled) return false;

    if (onClick) {
      onClick(evt);
    }

    return false;
  }

  render () {
    const { handleClick } = this;
    const { className, disabled, text, type, variant } = this.props;
    const fullClassName = classnames(`${baseClass}__${variant}`, className, {
      [baseClass]: variant !== 'unstyled',
      [`${baseClass}__${variant}--disabled`]: disabled,
    });

    return (
      <button
        className={fullClassName}
        disabled={disabled}
        onClick={handleClick}
        type={type}
      >
        {text}
      </button>
    );
  }
}

export default Button;

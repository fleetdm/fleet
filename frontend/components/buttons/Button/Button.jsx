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

    onClick(evt);

    return false;
  }

  render () {
    const { handleClick } = this;
    const { className, text, type, variant } = this.props;
    const fullClassName = classnames(baseClass, `${baseClass}__${variant}`, className);

    return (
      <button
        className={fullClassName}
        onClick={handleClick}
        type={type}
      >
        {text}
      </button>
    );
  }
}

export default Button;

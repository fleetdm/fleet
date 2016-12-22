import React, { Component, PropTypes } from 'react';
import { noop } from 'lodash';
import classnames from 'classnames';

import Icon from 'components/icons/Icon';

class Rocker extends Component {

  static propTypes = {
    className: PropTypes.string,
    onChange: PropTypes.func,
    options: PropTypes.shape({
      rightText: PropTypes.string,
      rightIcon: PropTypes.string,
      leftText: PropTypes.string,
      leftIcon: PropTypes.string,
    }),
    value: PropTypes.string,
  };

  static defaultProps = {
    onChange: noop,
  };

  handleChange = (evt) => {
    const { onChange, options: { rightText, leftText }, value } = this.props;
    evt.preventDefault();

    const newOption = value === leftText ? rightText : leftText;

    onChange(newOption);
  };

  render () {
    const { handleChange } = this;
    const { className, options, value } = this.props;
    const { rightText, rightIcon, leftText, leftIcon } = options;
    const baseClass = 'kolide-rocker';

    const rockerClasses = classnames(baseClass, className);
    const buttonClasses = classnames(`${baseClass}__button`, 'button', 'button--unstyled', {
      [`${baseClass}__button--checked`]: value === leftText,
    });

    return (
      <div className={rockerClasses}>
        <button className={buttonClasses} onClick={handleChange}>
          <span className={`${baseClass}__switch ${baseClass}__switch--left`}>
            <span className={`${baseClass}__text`}>
              <Icon name={leftIcon} /> {leftText}
            </span>
          </span>
          <span className={`${baseClass}__switch ${baseClass}__switch--right`}>
            <span className={`${baseClass}__text`}>
              <Icon name={rightIcon} /> {rightText}
            </span>
          </span>
        </button>
      </div>
    );
  }
}

export default Rocker;

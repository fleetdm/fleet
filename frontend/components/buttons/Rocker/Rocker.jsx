import React, { Component, PropTypes } from 'react';
import { noop } from 'lodash';
import classnames from 'classnames';

class Rocker extends Component {

  static propTypes = {
    className: PropTypes.string,
    handleChange: PropTypes.func,
    name: PropTypes.string,
    options: PropTypes.shape({
      aText: PropTypes.string,
      aIcon: PropTypes.string,
      bText: PropTypes.string,
      bIcon: PropTypes.string,
    }),
    value: PropTypes.string,
  };

  static defaultProps = {
    handleChange: noop,
  };

  render () {
    const { className, handleChange, name, options, value } = this.props;
    const { aText, aIcon, bText, bIcon } = options;
    const baseClass = 'kolide-rocker';

    const rockerClasses = classnames(baseClass, className);

    return (
      <div className={rockerClasses}>
        <label className={`${baseClass}__label`} htmlFor={name}>
          <input className={`${baseClass}__checkbox`} type="checkbox" value={value} name={name} id={name} onChange={handleChange} checked={value === bText} />
          <span className={`${baseClass}__switch ${baseClass}__switch--opt-b`}>
            <span className={`${baseClass}__text`}>
              <i className={`kolidecon kolidecon-${bIcon}`} /> {bText}
            </span>
          </span>
          <span className={`${baseClass}__switch ${baseClass}__switch--opt-a`}>
            <span className={`${baseClass}__text`}>
              <i className={`kolidecon kolidecon-${aIcon}`} /> {aText}
            </span>
          </span>
        </label>
      </div>
    );
  }
}

export default Rocker;

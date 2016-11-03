import React, { Component, PropTypes } from 'react';
import { Link } from 'react-router';
import classnames from 'classnames';

class StackedWhiteBoxes extends Component {
  static propTypes = {
    children: PropTypes.element,
    headerText: PropTypes.string,
    className: PropTypes.string,
    leadText: PropTypes.string,
    previousLocation: PropTypes.string,
  };

  renderBackButton = () => {
    const { previousLocation } = this.props;
    const baseClass = 'stack-box-back';

    if (!previousLocation) return false;

    return (
      <div className={baseClass}>
        <Link to={previousLocation} className={`${baseClass}__link`}>╳</Link>
      </div>
    );
  }

  renderHeader = () => {
    const { headerText, className } = this.props;
    const baseClass = 'stacked-box-header';

    const boxHeaderClass = classnames(
      baseClass,
      className
    );

    return (
      <div className={boxHeaderClass}>
        <p className={`${baseClass}__text`}>{headerText}</p>
      </div>
    );
  }

  render () {
    const { children, leadText } = this.props;
    const { renderBackButton, renderHeader } = this;

    const baseClass = 'stacked-white-boxes';

    return (
      <div className={baseClass}>
        <div className={`${baseClass}__box`}>
          {renderBackButton()}
          {renderHeader()}
          <p className={`${baseClass}__text`}>{leadText}</p>
          {children}
        </div>
      </div>
    );
  }
}

export default StackedWhiteBoxes;

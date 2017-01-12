import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';
import { round } from 'lodash';

const baseClass = 'progress-bar';

class ProgressBar extends Component {
  static propTypes = {
    className: PropTypes.string,
    max: PropTypes.number.isRequired,
    value: PropTypes.number.isRequired,
  };

  render () {
    const { className, max, value } = this.props;
    const percentComplete = `${(round((value / (max || 1)) * 100, 0))}%`;
    const wrapperClassName = classnames(baseClass, className);

    return (
      <div className={wrapperClassName}>
        <div style={{ width: percentComplete }} />
      </div>
    );
  }
}

export default ProgressBar;

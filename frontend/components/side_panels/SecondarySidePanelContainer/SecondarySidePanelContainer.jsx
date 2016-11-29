import React, { Component, PropTypes } from 'react';

class SecondarySidePanelContainer extends Component {
  static propTypes = {
    children: PropTypes.node,
    className: PropTypes.string,
  };

  render () {
    const { children, className } = this.props;

    return (
      <div className={`secondary-side-panel-container ${className}`}>
        {children}
      </div>
    );
  }
}

export default SecondarySidePanelContainer;

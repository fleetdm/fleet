import React, { Component, PropTypes } from 'react';

class SecondarySidePanelContainer extends Component {
  static propTypes = {
    children: PropTypes.node,
    className: PropTypes.string,
  };

  render () {
    const { children, className } = this.props;

    return (
      <div className={`${className} secondary-side-panel-container`}>
        {children}
      </div>
    );
  }
}

export default SecondarySidePanelContainer;

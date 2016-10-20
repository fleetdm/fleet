import React, { Component, PropTypes } from 'react';
import radium from 'radium';

import componentStyles from './styles';

class SecondarySidePanelContainer extends Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render () {
    const { children } = this.props;
    const { containerStyles } = componentStyles;

    return (
      <div style={containerStyles}>
        {children}
      </div>
    );
  }
}

export default radium(SecondarySidePanelContainer);

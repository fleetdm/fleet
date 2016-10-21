import React, { Component, PropTypes } from 'react';
import radium from 'radium';

import componentStyles from './styles';

class Modal extends Component {
  static propTypes = {
    children: PropTypes.node,
    onExit: PropTypes.func,
    title: PropTypes.string,
  };

  render () {
    const { children, onExit, title } = this.props;
    const {
      containerStyles,
      contentStyles,
      exStyles,
      headerStyles,
      modalStyles,
    } = componentStyles;

    return (
      <div style={containerStyles}>
        <div style={modalStyles}>
          <div style={headerStyles}>
            <span>{title}</span>
            <button className="btn--unstyled" style={exStyles} onClick={onExit}>╳</button>
          </div>
          <div style={contentStyles}>
            {children}
          </div>
        </div>
      </div>
    );
  }
}

export default radium(Modal);

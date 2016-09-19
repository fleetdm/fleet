import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import componentStyles from './styles';

class AuthenticationFormWrapper extends Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render () {
    const { children } = this.props;
    const { containerStyles, whiteTabStyles } = componentStyles;

    return (
      <div style={containerStyles}>
        <div style={whiteTabStyles} />
        {children}
      </div>
    );
  }
}

export default radium(AuthenticationFormWrapper);
